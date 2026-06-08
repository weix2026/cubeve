use clap::Parser;

#[derive(Parser)]
#[command(name = "cubeve-version")]
#[command(about = "CubeVE Version Tool")]
#[command(version = env!("CARGO_PKG_VERSION"))]
struct Args {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(clap::Subcommand)]
enum Commands {
    /// Show version information
    Version,
    /// Show detailed build info
    BuildInfo,
}

fn main() {
    let args = Args::parse();

    match args.command {
        Some(Commands::Version) | None => {
            println!("CubeVE Version: {}", env!("CARGO_PKG_VERSION"));
            println!("Virtual Infrastructure Platform");
            println!("https://github.com/weix2026/cubeve");
        }
        Some(Commands::BuildInfo) => {
            println!("CubeVE Version: {}", env!("CARGO_PKG_VERSION"));
            println!("Build Time: {}", option_env!("BUILD_TIME").unwrap_or("unknown"));
            println!("Git Commit: {}", option_env!("GIT_COMMIT").unwrap_or("unknown"));
        }
    }
}
