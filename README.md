# pipe

- convert and transfer data flow between different protocols

## workflow

### create your own packer to encrypt & decrypt your protocol's data flow

```golang
key := make([]byte, 32)
iv := make([]byte, 16)
rand.Read(key)
rand.Read(iv)
packer := &pipe.AESPacker{
    Key: key,
    IV:  iv,
}
```

**notice**: the packer is just optional, if you don't need to encrypt your data but just need to transfer the data by another protocol, just leave it empty

###  transfer your data flow by another protocol
- client: deploy this pipe client to be a proxy of your real client
```golang
localAddr := "localhost:8080"
remoteAddr := "localhost:8081"
pClient := &pipe.Pipe{
    Listen: pipe.ListenUDP(localAddr),
    Dial: func() (net.Conn, error) {
        return net.Dial("tcp", remoteAddr)
    },
    Pack:   packer.CBCEncrypt,
    Unpack: packer.CBCDecrypt,
    Timeout: 60 * time.Second,
}
pClient.Start()
```

- server: deploy this pipe server to be a reverse proxy of your real server
```golang
localAddr := "localhost:8081"
remoteAddr := "localhost:8082"
pServer := &pipe.Pipe{
    Listen: func() (net.Listener, error) {
        return net.Listen("tcp", localAddr)
    },
    Dial:   pipe.DialUDP(remoteAddr),
    Pack:   packer.CBCDecrypt,
    Unpack: packer.CBCEncrypt,
    Timeout: 60 * time.Second,
}
pServer.Start()
```
