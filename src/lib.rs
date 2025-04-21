use std::env;
use std::error::Error;
use std::fs::{self, File};
use std::io::{BufRead, BufReader};
use std::path::Path;

const BOLD_WHITE: &str = "\x1b[1;37m";
const FADED: &str = "  \x1b[2m";
const RESET: &str = "\x1b[0m";

pub struct Sheet {
    pub name: String,
    pub filepath: String,
}

impl Sheet {
    pub fn new(name: String) -> Result<Sheet, Box<dyn Error>> {
        let home_path = env::var("HOME")
            .map_err(|e| format!("Failed to get HOME environment variable: {}", e))?;

        let sheets_dir = format!("{}/.sheets", home_path);
        let dir_exists = Path::new(&sheets_dir).try_exists().is_ok();

        if !dir_exists {
            fs::create_dir(&sheets_dir)
                .map_err(|e| format!("Failed to create the {} directory: {}", sheets_dir, e))?
        }

        let filepath = format!("{}/{}", sheets_dir, name);

        Ok(Sheet { name, filepath })
    }

    pub fn parse(&self) -> Result<(), Box<dyn Error>> {
        let file = File::open(&self.filepath)?;
        let reader = BufReader::new(file);

        println!("\n");
        for line in reader.lines() {
            let line = line?;
            match line {
                line if line.starts_with('#') => println!("{BOLD_WHITE}{line}{RESET}"),
                line if line.contains("//") => {
                    let formatted = line.replace("//", FADED);
                    println!("{formatted}{RESET}");
                }
                _ => println!("{line}"),
            }
        }
        println!("\n");

        Ok(())
    }
}
