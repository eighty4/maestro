use std::io::Write;
use std::net::{Shutdown, TcpListener};

fn main() {
    let listener = TcpListener::bind(("127.0.0.1", 8001)).unwrap();
    for stream in listener.incoming() {
        match stream {
            Ok(mut tcp_stream) => {
                tcp_stream.write(b"hello").expect("tcp write");
                tcp_stream.flush().expect("tcp flush");
                tcp_stream.shutdown(Shutdown::Both).expect("tcp close");
            }
            Err(e) => {
                println!("tcp connection error: {}", e);
            }
        }
    }
}
