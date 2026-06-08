use clap::{Parser, Subcommand};
use anyhow::Result;

#[derive(Parser)]
#[command(name = "cubeconsole")]
#[command(about = "CubeVE Console - CLI for Virtual Infrastructure")]
#[command(version = env!("CARGO_PKG_VERSION"))]
struct Args {
    #[command(subcommand)]
    command: Commands,

    /// API Gateway endpoint
    #[arg(short, long, default_value = "http://localhost:8080")]
    api: String,
}

#[derive(Subcommand)]
enum Commands {
    /// Instance operations
    Instance {
        #[command(subcommand)]
        command: InstanceCommands,
    },
    /// Show version
    Version,
}

#[derive(Subcommand)]
enum InstanceCommands {
    /// List all instances
    List,
    /// Get instance details
    Show { name: String },
    /// Create new instance
    Create {
        name: String,
        #[arg(short, long)]
        image: Option<String>,
    },
    /// Delete instance
    Delete { name: String },
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();

    match args.command {
        Commands::Instance { command } => {
            let client = reqwest::Client::new();
            match command {
                InstanceCommands::List => {
                    let resp = client
                        .get(format!("{}/api/v1/instances", args.api))
                        .send()
                        .await?;
                    let text = resp.text().await?;
                    println!("{}", text);
                }
                InstanceCommands::Show { name } => {
                    let resp = client
                        .get(format!("{}/api/v1/instances/{}", args.api, name))
                        .send()
                        .await?;
                    let text = resp.text().await?;
                    println!("{}", text);
                }
                InstanceCommands::Create { name, image } => {
                    let body = serde_json::json!({
                        "name": name,
                        "source": {
                            "type": "image",
                            "alias": image.unwrap_or_else(|| "ubuntu/24.04".to_string()),
                        },
                        "type": "container",
                    });
                    let resp = client
                        .post(format!("{}/api/v1/instances", args.api))
                        .json(&body)
                        .send()
                        .await?;
                    let text = resp.text().await?;
                    println!("{}", text);
                }
                InstanceCommands::Delete { name } => {
                    let resp = client
                        .delete(format!("{}/api/v1/instances/{}", args.api, name))
                        .send()
                        .await?;
                    let text = resp.text().await?;
                    println!("{}", text);
                }
            }
        }
        Commands::Version => {
            println!("cubeve version {}", env!("CARGO_PKG_VERSION"));
            println!("Virtual Infrastructure Platform");
        }
    }

    Ok(())
}
