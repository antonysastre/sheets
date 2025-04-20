use std::env;
use std::process;

use sheets::Sheet;

fn main() {
    let args: Vec<String> = env::args().collect();

    match Sheet::new(&args) {
        Ok(sheet) => {
            sheet.parse().expect("Couldn't parse the sheet.");
            process::exit(0);
        }
        Err(e) => {
            println!("Problem parsing command: {}", e);
            process::exit(1);
        }
    };
}
