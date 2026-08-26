基于 Go 实现的 OrbitLink 项目，一款卫星地面站过境接收编排服务，协调天线所有权、射频计划、接收机捕获和遥测记录。

OrbitLink coordinates satellite pass reception, antenna ownership, RF plans, receiver acquisition and durable frame recording for a ground station.

Run locally with Go 1.26.2:

```text
go run ./cmd/orbitlink -listen 127.0.0.1:19705 -state-root ./var/orbitlink
```

Open `/passes`, `/antennas`, `/rf-chains` and `/recordings` for the four operations views. The service exposes `/healthz` and matching JSON APIs under `/api`.
