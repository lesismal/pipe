# pipe

- convert and transfer data flow between different protocols

## workflow

### create your own packer to encrypt & decrypt your protocol's data flow

```golang
import "github.com/lesismal/pipe/packer"

key := make([]byte, 32)
iv := make([]byte, 16)
rand.Read(key)
rand.Read(iv)
packer := &packer.AESCBC{
    Key: key,
    IV:  iv,
}
```

**notice**: the packer is just optional, if you don't need to encrypt your data but just need to transfer the data by another protocol, just leave it empty

###  transfer your data flow by another protocol
- client: deploy this pipe client to be a proxy of your real client
```golang
import "github.com/lesismal/pipe/protocol"

localAddr := "localhost:8080"
remoteAddr := "localhost:8081"
pClient := &pipe.Pipe{
    Listen: protocol.ListenUDP(localAddr),
    Dial: protocol.DialTCP(remoteAddr)
    Packer:   packer,
    Timeout: 60 * time.Second,
}
pClient.Start()
```

- server: deploy this pipe server to be a reverse proxy of your real server
```golang
import "github.com/lesismal/pipe/protocol"

localAddr := "localhost:8081"
remoteAddr := "localhost:8082"
pServer := &pipe.Pipe{
    Listen: protocol.ListenTCP(localAddr),
    Dial: protocol.DialUDP(remoteAddr)
    Pack:   packer.CBCDecrypt,
    Unpack: packer.CBCEncrypt,
    Timeout: 60 * time.Second,
}
pServer.Start()
```
