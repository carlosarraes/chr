use clap::{Parser, Subcommand};
use dialoguer::Input;
use std::process::Command;

#[derive(Parser)]
#[command(
    name = "chr",
    version = "1.0",
    about = "A simple CLI tool to manage main and homolog braches"
)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Start,
}

fn main() {
    let args = Cli::parse();

    match args.command {
        Commands::Start => {
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
