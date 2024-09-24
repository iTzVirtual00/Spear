# Project: Spear - Peer-to-Peer File Transfer Tool

Welcome to Spear, my first project in Go! This command-line tool is designed for sending and receiving files peer-to-peer with encryption. I created this project to learn Go.

## Architecture

Spear consists of a sender and a receiver, both of which can act as either a server or a client. The sender initiates the connection to the receiver or vice versa. Each server generates its certificates on the fly and saves them as `private.pem` and `cert.pem`.

## Protocol

The protocol relies on encrypted WebSocket messages, leveraging the [gorilla/websocket](https://github.com/gorilla/websocket) library. Here's an overview:

- The first WebSocket handshake establishes the connection.
- Subsequent handshakes are discarded.
- The first message sent is the expected file length.
- Following messages contain the file content, divided into 1024-byte chunks.
- The connection closes after the last byte is sent.

## Commands

To use Spear, you can execute the following commands:

- **Receive files**:
  ```
  spear receive -a 127.0.0.1:8080 -f receivedfile.txt
  ```
  or
  ```
  spear receive -l 8080 -f receivedfile.txt
  ```

- **Send files**:
  ```
  spear send -l 8080 -f filetosend.txt
  ```
  or
  ```
  spear send -a 127.0.0.1:8080 -f filetosend.txt
  ```

## TODOs

- Improve certificate management for enhanced security.
- Implement client certificate authentication by the server to accept only trusted clients.
- Refactor the codebase to make it more modular and maintainable.
- Better error handling.
- Add support for sending multiple files in a single session.
- Enable interactive client acceptance via the console.

Feel free to contribute to Spear or provide feedback
