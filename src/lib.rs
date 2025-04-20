use std::error::Error;
use std::fs::{self, File};
use std::io::{BufRead, BufReader};
use std::{env, path};

const BOLD_WHITE: &str = "\x1b[1;37m";
const FADED: &str = "  \x1b[2m";
const RESET: &str = "\x1b[0m";

pub struct Sheet {
    pub name: String,
    pub filepath: String,
}

impl Sheet {
    pub fn new(name: &str) -> Result<Sheet, &str> {
        let home_path = match env::var("HOME") {
            Ok(path) => path,
            Err(_) => return Err("Failed to get HOME environment variable"),
        };

        let sheets_dir = format!("{}/.sheets", home_path);
        let dir_exists = path::Path::new(&sheets_dir).exists();

        if !dir_exists {
            match fs::create_dir(&sheets_dir) {
                Ok(_) => (),
                Err(_) => return Err("Failed to create .sheets directory"),
            }
        }

        let filepath = format!("{}/{}", sheets_dir, name);

        Ok(Sheet {
            filepath,
            name: name.to_string(),
        })
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
