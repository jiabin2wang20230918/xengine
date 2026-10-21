# xengine

Offline **SceneEngine** for PLC gateway: scene / shadow / compare / linkage / taskconf / watch.

Originally developed as `gitlab.biot.tcl.com/embedded/xengine`. This public module is the same SceneEngine surface used by [plc-gateway-core](https://github.com/jiabin2wang20230918/plc-gateway-core) for local (offline) scene recall — property/timer triggers, device shadow, compare conditions, and linkage rules.

## Packages

| Package | Role |
|---------|------|
| `scene` | Scene evaluation (triggers, conditions, actions) |
| `shadow` | Device property shadow / cache |
| `compare` | Condition comparison helpers |
| `linkage` | Linkage rules + optional GORM persistence |
| `taskconf` | Task / protocol config manager |
| `watch` | Property watch → scene evaluation |
| `utils` | Shared consts, logging, beehive message helpers |

## Install

```bash
go get github.com/jiabin2wang20230918/xengine@v0.1.0
```

## License

MIT
