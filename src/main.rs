use clap::{Parser, Subcommand};
use std::process::{Command, Stdio};
use std::fs;
use serde::{Deserialize, Serialize};
use dialoguer::Input;
use anyhow::{Context, Result, anyhow, bail};

#[derive(Parser)]
#[command(
    name = "chr",
    version = "0.0.2",
    about = "A simple CLI tool to manage braches and commits",
    long_about = "A simple CLI tool to manage branches and commits.\nFor more information, try '--help'."
)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Parser)]
struct PickArgs {
    #[arg(short, long, default_value = "5", help = "Number of commits to pick")]
    count: u32,
    #[arg(
        short,
        long,
        help = "Pick latest commits from the current user\n*Rebases might give you already picked commits"
    )]
    latest: bool,
    #[arg(short, long, help = "Show commits instead of picking")]
    show: bool,
}

#[derive(Subcommand)]
enum Commands {
    #[command(
        about = "Start a new card branch.",
        long_about = "Start a new card branch.\n\nThis command checks the repository status (unless debug mode is enabled), prompts for a card number, and then creates a new branch following the pattern 'ZUP-<card_number>-prd'."
    )]
    Pick(PickArgs),
    #[command(
        about = "Create or update configuration file",
        long_about = "Create or update configuration file at ~/.config/chr.toml with custom prefix and suffixes."
    )]
    Config,
}

const DEFAULT_PREFIX: &str = "ZUP-";
const DEFAULT_SUFFIX_PRD: &str = "-prd";
const DEFAULT_SUFFIX_HML: &str = "-hml";

#[derive(Deserialize, Serialize, Debug, Default)]
struct Config {
    prefix: Option<String>,
    suffix_prd: Option<String>,
    suffix_hml: Option<String>,
}

fn load_config() -> Config {
    let config_path = dirs::home_dir()
        .unwrap_or_default()
        .join(".config")
        .join("chr.toml");
    
    if config_path.exists() {
        match fs::read_to_string(&config_path) {
            Ok(contents) => {
                match toml::from_str(&contents) {
                    Ok(config) => return config,
                    Err(e) => eprintln!("Error parsing config file: {}", e),
                }
            },
            Err(e) => eprintln!("Error reading config file: {}", e),
        }
    }
    
    Config::default()
}

fn main() {
    let args = Cli::parse();
    
    let result = match args.command {
        Commands::Pick(pick_args) => pick(pick_args),
        Commands::Config => create_config(),
    };

    if let Err(e) = result {
        eprintln!("Error: {:#}", e);
        std::process::exit(1);
    }
}

fn pick(args: PickArgs) -> Result<()> {
    let config = load_config();
    let prefix = config.prefix.as_deref().unwrap_or(DEFAULT_PREFIX);
    let suffix_prd = config.suffix_prd.as_deref().unwrap_or(DEFAULT_SUFFIX_PRD);
    let suffix_hml = config.suffix_hml.as_deref().unwrap_or(DEFAULT_SUFFIX_HML);

    let branch_output = Command::new("git")
        .arg("branch")
        .arg("--show-current")
        .output()
        .context("Failed to get current branch name")?;
    
    let branch_name = String::from_utf8(branch_output.stdout)
        .context("Failed to parse branch name")?
        .trim()
        .to_string();
    
    let parts: Vec<&str> = branch_name.split("-").collect();
    if parts.len() < 2 {
        bail!("Current branch '{}' doesn't match the expected format '{}<card-number>{}'", 
            branch_name, prefix, suffix_prd);
    }
    
    if !branch_name.starts_with(prefix) {
        bail!("Current branch '{}' doesn't start with the expected prefix '{}'\nExpected format: '{}<card-number>{}'", 
            branch_name, prefix, prefix, suffix_prd);
    }
    
    let card_number = parts.get(1).ok_or_else(|| 
        anyhow!("Could not extract card number from branch name '{}'", branch_name)
    )?;

    let hml_branch = format!("{}{}{}", prefix, card_number, suffix_hml);
    let prd_branch = format!("{}{}{}", prefix, card_number, suffix_prd);

    let branch_exists = |branch: &str| -> Result<bool> {
        let output = Command::new("git")
            .arg("rev-parse")
            .arg("--verify")
            .arg(branch)
            .output()
            .context(format!("Failed to check if branch '{}' exists", branch))?;
        Ok(output.status.success())
    };
    
    if !branch_exists(&prd_branch)? {
        bail!("Production branch '{}' does not exist", prd_branch);
    }
    
    if !branch_exists(&hml_branch)? {
        bail!("Homologation branch '{}' does not exist", hml_branch);
    }

    let commit_count = if args.latest { 100 } else { args.count };

    let log_output = Command::new("git")
        .arg("log")
        .arg(format!("^{}", &hml_branch))
        .arg(&prd_branch)
        .arg(format!("-{}", commit_count))
        .arg("--format=%h|%an|%s")
        .output()
        .context("Failed to execute git log")?;
    
    if !log_output.status.success() {
        bail!("Failed to get commit logs. Make sure both branches exist.");
    }
    
    let output = String::from_utf8(log_output.stdout)
        .context("Failed to parse git log output")?;

    let current_user = get_current_user()?;

    let final_lines: Vec<&str> = if args.latest {
        output
            .lines()
            .filter(|line| {
                let parts: Vec<&str> = line.split("|").collect();
                parts.len() >= 3 && parts[1].trim() == current_user
            })
            .collect()
    } else {
        output.lines().collect()
    };

    if final_lines.is_empty() {
        if args.latest {
            println!("No commits found for user '{}'", current_user);
        } else {
            println!("No commits found between '{}' and '{}'", &hml_branch, &prd_branch);
        }
        return Ok(());
    }

    for line in &final_lines {
        let parts: Vec<&str> = line.split("|").collect();
        if parts.len() < 3 {
            println!("{}", line);
        } else {
            let commit = parts[0].trim();
            let author = parts[1].trim();
            let message = parts[2].trim();
            let colored_author = if author == current_user {
                format!("\x1b[32m{}\x1b[0m", author)
            } else {
                format!("\x1b[31m{}\x1b[0m", author)
            };
            println!("{} | {} | {}", commit, colored_author, message);
        }
    }

    let commit_hashes: Vec<&str> = final_lines
        .iter()
        .filter_map(|line| {
            let parts: Vec<&str> = line.split("|").collect();
            if parts.len() >= 3 {
                Some(parts[0].trim())
            } else {
                None
            }
        })
        .collect();

    if commit_hashes.is_empty() {
        println!("No valid commit hashes found in the output");
        return Ok(());
    }

    if args.show {
        return Ok(());
    }

    let ques = dialoguer::Confirm::new()
        .with_prompt("Do you want to cherry-pick these commits?")
        .interact()
        .context("Failed to get user confirmation")?;
        
    if ques {
        let oldest_commit = commit_hashes.last()
            .ok_or_else(|| anyhow!("No commits to cherry-pick"))?;
            
        let newest_commit = commit_hashes.first()
            .ok_or_else(|| anyhow!("No commits to cherry-pick"))?;
            
        let range = format!("{}^..{}", oldest_commit, newest_commit);

        let rev_process = Command::new("git")
            .arg("rev-list")
            .arg("--reverse")
            .arg(&range)
            .stdout(Stdio::piped())
            .spawn()
            .context("Failed to execute git rev-list")?;
            
        let rev_stdout = rev_process.stdout
            .ok_or_else(|| anyhow!("Failed to capture git rev-list output"))?;

        let status = Command::new("git")
            .arg("cherry-pick")
            .arg("--stdin")
            .stdin(rev_stdout)
            .status()
            .context("Failed to execute git cherry-pick")?;
            
        if status.success() {
            println!("Successfully cherry-picked commits");
        } else {
            println!("Cherry-pick operation failed. You may need to resolve conflicts.");
        }
    }
    
    Ok(())
}

fn get_current_user() -> Result<String> {
    let output = Command::new("git")
        .arg("config")
        .arg("user.name")
        .output()
        .context("Failed to get git user name")?;
        
    let user = String::from_utf8(output.stdout)
        .context("Failed to parse git user name")?
        .trim()
        .to_string();
        
    Ok(user)
}

fn create_config() -> Result<()> {
    let config_dir = dirs::home_dir()
        .ok_or_else(|| anyhow!("Failed to determine home directory"))?
        .join(".config");
    
    let config_path = config_dir.join("chr.toml");
    
    if !config_dir.exists() {
        fs::create_dir_all(&config_dir)
            .context(format!("Failed to create directory: {}", config_dir.display()))?;
        println!("Created directory: {}", config_dir.display());
    }

    let current_config = load_config();
    
    let prefix: String = Input::new()
        .with_prompt("Enter prefix for branch names")
        .default(current_config.prefix.unwrap_or_else(|| DEFAULT_PREFIX.to_string()))
        .interact()
        .context("Failed to get prefix input")?;
    
    let suffix_prd: String = Input::new()
        .with_prompt("Enter suffix for production branches")
        .default(current_config.suffix_prd.unwrap_or_else(|| DEFAULT_SUFFIX_PRD.to_string()))
        .interact()
        .context("Failed to get production suffix input")?;
    
    let suffix_hml: String = Input::new()
        .with_prompt("Enter suffix for homologation branches")
        .default(current_config.suffix_hml.unwrap_or_else(|| DEFAULT_SUFFIX_HML.to_string()))
        .interact()
        .context("Failed to get homologation suffix input")?;
    
    let new_config = Config {
        prefix: Some(prefix),
        suffix_prd: Some(suffix_prd),
        suffix_hml: Some(suffix_hml),
    };
    
    let toml_string = toml::to_string(&new_config)
        .context("Failed to convert configuration to TOML")?;
    
    let config_content = format!(
        "# Configuration file for chr tool\n\
        # Generated by 'chr config' command\n\n\
        # The prefix for branch names (default: \"{}\")\n\
        {}\n\
        # The suffix for production branches (default: \"{}\")\n\
        {}\n\
        # The suffix for homologation branches (default: \"{}\")\n\
        {}\n",
        DEFAULT_PREFIX,
        toml_string.lines().find(|l| l.starts_with("prefix")).unwrap_or("prefix = \"\""),
        DEFAULT_SUFFIX_PRD,
        toml_string.lines().find(|l| l.starts_with("suffix_prd")).unwrap_or("suffix_prd = \"\""),
        DEFAULT_SUFFIX_HML,
        toml_string.lines().find(|l| l.starts_with("suffix_hml")).unwrap_or("suffix_hml = \"\"")
    );
    
    fs::write(&config_path, config_content)
        .context(format!("Failed to write configuration to {}", config_path.display()))?;
        
    println!("Configuration written to {}", config_path.display());
    
    Ok(())
}
