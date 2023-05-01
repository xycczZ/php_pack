## php_pack

### usage
```go
r, err := pack.PHPPack("c2n2", 0x1234, 0x5678, 65, 66)
m, err := unpack.PHPUnpack(unpack.NewOption("c2chars/n2int", r))
// map[chars1:52 chars2:120 int1:65 int2:66]
```