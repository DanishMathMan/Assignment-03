# Distributed Systems, BSc (Autumn 2025) - Assignment 3 
## by freju@itu.dk, frvs@itu.dk and alyp@itu.dk

# Real-time Chat Application
### Prerequisites:
- Golang (1.25+)
- Basic knowledge about running go programs.

## How to run:

1. Navigate to the directory where you've downloaded the github repository\
   ```cd <directoryOfRepository>/server```
2. Run the server: \
   ```go run server.go```
3. Navigate to the client directory: \
```cd <directoryOfRepository>/client```
4. Run the client: \
   ```go run client.go```

You can run as many clients as you want, as long as the server is open. \
ENJOY!

## How to stop the program:
### Server:
1. Navigate to the terminal where your server process is running
2. Press ```ctrl + c``` (note it might take a few seconds for the server to shut down properly)

### Client:
1. Navigate to the terminal where your client process is running
2. Either: \
   2.1: Press ```ctrl + c``` \
   2.2: Type: ```--exit``` (this is the preferred method of exiting the program)

## Client:

When running as a client, you will first be prompted to choose a username.

Then the program will make a connection to the server and start the chat.

Once you are in the chat, you will see messages from other users of the form:
```txt
[<user> at LT: <logical_timestamp>] - <message> 
```

You can write messages simply by typing them in the terminal.\
NOTE: there exists a visual bug when writing a message while receiving a message from another user.
This is purely visual, and you can still write a message, and the message you are typing will simply display awkwardly while typing.

## Logging

Logging is done differently for server and for client.

### Server logging

The server will log received messages, broadcasted messages, connection and disconnection events, as well as 
server shutdown and startup. These are associated with logical timestamps and are output to the folder ServerLogs with a 
name of the type "log_\<time>.txt". where \<times> is the local time of server when the program is run.

### Client logging
The client only logs messages sent and received and when it connects and disconnects. These are associated with logical timestamps and are output to the folder ClientLogs with a 
name of the type "log_\<time>.txt". where \<times> is the local time of a client when the program is run.
































