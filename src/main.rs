use clap::{Parser, Subcommand};
use dialoguer::Input;
use std::process::{Command, Stdio};

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
    Start,
    #[command(
        about = "Show the last 5 commits from the PRD and HML branches.",
        long_about = "Show the last 5 commits from the PRD and HML branches.\n\nThis command shows the last 5 commits from the PRD and HML branches of the current card branch."
    )]
    Pick(PickArgs),
}

const PREFIX: &str = "ZUP-";
const SUFFIX_PRD: &str = "-prd";
const SUFFIX_HML: &str = "-hml";

fn main() {
    let args = Cli::parse();

    match args.command {
        Commands::Start => start(),
        Commands::Pick(pick_args) => pick(pick_args),
    }
}

fn pick(args: PickArgs) {
    let branch_output = Command::new("git")
        .arg("branch")
        .arg("--show-current")
        .output()
        .expect("Failed to get current branch name")
        .stdout;
    let branch_name = String::from_utf8(branch_output).unwrap();
    let branch_name = branch_name.trim();
    let parts = branch_name.split("-").collect::<Vec<&str>>();
    let card_number = parts[1];

    let hml_branch = format!("{}{}{}", PREFIX, card_number, SUFFIX_HML);
    let prd_branch = format!("{}{}{}", PREFIX, card_number, SUFFIX_PRD);

    let log_output = Command::new("git")
        .arg("log")
        .arg(format!("^{}", hml_branch))
        .arg(prd_branch)
        .arg(format!("-{}", args.count))
        .arg("--format=%h %C(green)%an%C(reset) %s")
        .arg("--color=always")
        .output()
        .expect("Failed to execute git log");

    let output = String::from_utf8(log_output.stdout).unwrap();

    if args.latest {
        let user_output = Command::new("git")
            .arg("config")
            .arg("user.name")
            .output()
            .expect("Failed to get git user name");
        let username = String::from_utf8(user_output.stdout)
            .unwrap()
            .trim()
            .to_string();

        for line in output.lines() {
            if line.contains(&username) {
                println!("{}", line);
            } else {
                break;
            }
        }
    } else {
        print!("{}", output);
    }

    if !args.show {
        let ques = dialoguer::Confirm::new()
            .with_prompt("Do you want to cherry-pick these commits?")
            .interact()
            .unwrap();

        if ques {
            let commit_hashes = output
                .lines()
                .map(|line| line.split_whitespace().collect::<Vec<&str>>()[0])
                .collect::<Vec<&str>>();

            let rev_output = Command::new("git")
                .arg("rev-list")
                .arg("--reverse")
                .arg(format!(
                    "{}^..{}",
                    commit_hashes[0],
                    commit_hashes[commit_hashes.len() - 1]
                ))
                .stdout(Stdio::piped())
                .spawn()
                .expect("Failed to execute git rev-list");

            let rev_stdout = rev_output.stdout.unwrap();

            Command::new("git")
                .arg("cherry-pick")
                .arg("--stdin")
                .stdin(rev_stdout)
                .status()
                .expect("Failed to execute git cherry-pick");
        }
    }
}

fn start() {
    let mut git = Command::new("git");

    let card_number: String = Input::new()
        .with_prompt("Card number?")
        .validate_with(|input: &String| -> Result<(), &str> {
            input
                .parse::<u32>()
                .map(|_| ())
                .map_err(|_| "Please enter a valid number")
        })
        .interact()
        .unwrap();

    let branch_name = format!("ZUP-{}-prd", card_number);

    git.arg("switch")
        .arg("main")
        .status()
        .expect("Failed to switch to main branch");

    git.arg("fetch")
        .status()
        .expect("Failed to execute git fetch");

    git.arg("pull")
        .status()
        .expect("Failed to execute git pull");

    git.arg("switch")
        .arg("-c")
        .arg(&branch_name)
        .status()
        .expect("Failed to create new branch");
}
