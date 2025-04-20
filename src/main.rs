use std::env;
use std::process;

use sheets::Sheet;

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        println!("No sheet given.\nCommand usage: `sheets <sheetname>`");
        process::exit(1);
    }

    match Sheet::new(&args[1]) {
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
