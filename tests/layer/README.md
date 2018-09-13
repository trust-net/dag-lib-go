## Test application for P2P Layer
Build and run the application as following:

#### build an instance

```
go build
```

#### run first instances

```
./layer config1.json
```

#### run second instance from a different terminal

```
./layer config2.json
```
#### let the two instances discover and connect with each other

```
P2P Layer enode://3fdb80a5d9bbdbea20a80aeb36ee49ca18267e151fd5ebb69ddb2af8c230688debea7f4cd077368a1050ee35c9005905380ddb0bb8d0e5aa3d101eb51ef785da@[::]:50438 created, waiting for a peer connection...
Connected with new peer: test node 1 Peer c3da24ed70538b73 127.0.0.1:50883
Waiting 5 seconds . . . . . .
Disconnecting with new peer: test node 1 Peer c3da24ed70538b73 127.0.0.1:50883
Peer connection successfully completed... Exiting.
```