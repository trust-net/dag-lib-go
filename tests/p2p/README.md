## Test application for P2P Layer
This is a test application for end to end testing of P2P layer library. This is a very simple app, that instantiates the P2P layer and connects with another peer on network running with same protocol/network. Goal is to test the peer discovery and peer connection.

Build and run the application as following:

#### build an instance

```
go build
```

#### run first instances

```
./p2p config1.json
```

#### run second instance from a different terminal

```
./p2p config2.json
```
#### let the two instances discover and connect with each other

```
P2P Layer enode://3fdb80a5d9bbdbea20a80aeb36ee49ca18267e151fd5ebb69ddb2af8c230688debea7f4cd077368a1050ee35c9005905380ddb0bb8d0e5aa3d101eb51ef785da@[::]:50438 created, waiting for a peer connection...
Connected with new peer: test node 1 Peer c3da24ed70538b73 127.0.0.1:50883
Waiting 5 seconds . . . . . .
Disconnecting with new peer: test node 1 Peer c3da24ed70538b73 127.0.0.1:50883
Peer connection successfully completed... Exiting.
```