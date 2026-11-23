# Sinh Tồn — prototype

Game sinh tồn 2D góc nhìn 3/4, viết bằng Go + [Ebitengine](https://ebitengine.org).
Tên và art đều là tạm; mọi hình vẽ được dựng bằng code trong `internal/client/art.go`.

## Chạy

Trên Windows (cần Go 1.25+, không cần cgo):

    go run ./cmd/game
    go run ./cmd/game -seed 42 -name Tien -size 128

Build `.exe` cho Windows từ Linux:

    make build-windows      # ra dist/survival-game.exe

Chạy client trên Linux cần cgo và các gói X11/GL dev (xem tài liệu cài đặt của Ebitengine).

## Điều khiển

| Phím | Tác dụng |
| --- | --- |
| WASD / mũi tên | Di chuyển |
| Giữ Space | Chặt cây, đập đá, hái quả, cắt cỏ, thêm củi vào lửa |
| E | Ăn quả mọng |
| C | Dựng lửa trại (2 Gỗ, 2 Cỏ khô, 3 Đá) |
| R | Hồi sinh khi đã chết |
| F11 | Toàn màn hình |

Ban đêm, đứng ngoài vùng sáng của lửa trại sẽ bị lạnh và mất máu.

## Kiến trúc

    cmd/game            client; chơi 1 người thì chạy luôn server trong cùng process
    internal/sim        luật chơi: sinh map, người chơi, tài nguyên, chế tạo, ngày/đêm
    internal/protocol   message giữa client và server
    internal/server     vòng tick 20Hz, kết nối, snapshot theo bán kính quan tâm
    internal/client     Ebitengine: input, vẽ, ánh sáng, HUD

- `sim` không biết gì về đồ hoạ hay mạng, chỉ tính theo tick, nên test được và chạy được trong server riêng.
- Server là nơi duy nhất quyết định trạng thái. Client gửi input, nhận snapshot và vẽ nội suy giữa hai snapshot gần nhất.
- Chơi 1 người vẫn đi qua đúng ranh giới client–server (`server.ConnectLocal`). Thêm co-op LAN là viết một transport TCP thoả `client.Conn`, không đụng tới luật chơi.

## Test

    make test    # sim, protocol, server (kèm race detector)
    make vet     # vet cả client cho Windows

`SIM_DUMP=1 go test ./internal/sim -run Dump -v` in map ra ASCII để chỉnh thông số sinh map.
