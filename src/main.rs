use clap::{Parser, Subcommand};
use dialoguer::Input;
use std::process::Command;

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

    #[arg(
        short,
        long,
        global = true,
        help = "Enable debug mode for more verbose output. When enabled, the command skips checking for uncommitted changes."
    )]
    debug: bool,
}

#[derive(Subcommand)]
enum Commands {
    #[command(
        about = "Start a new card branch.",
        long_about = "Start a new card branch.\n\nThis command checks the repository status (unless debug mode is enabled), prompts for a card number, and then creates a new branch following the pattern 'ZUP-<card_number>-prd'."
    )]
    Start,
}

fn main() {
    let args = Cli::parse();
    let mut git = Command::new("git");

    match args.command {
        Commands::Start => {
            if !args.debug {
                if !git
                    .args(["status", "--porcelain"])
                    .output()
                    .expect("Failed to execute git status")
                    .stdout
                    .is_empty()
                {
                    println!(
                    "You have uncommited changes. Please commit them before starting a new card"
                );
                    std::process::exit(1);
                }
            }

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

            git.args(["switch", "main"])
                .status()
                .expect("Failed to switch to main branch");

            Command::new("git")
                .arg("fetch")
                .status()
                .expect("Failed to execute git fetch");

            Command::new("git")
                .arg("pull")
                .status()
                .expect("Failed to execute git pull");

            Command::new("git")
                .args(["switch", "-c", &branch_name])
                .status()
                .expect("Failed to create new branch");
        }
    }
}
