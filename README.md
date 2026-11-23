# Sinh Tồn — prototype

Game sinh tồn 2D góc nhìn 3/4, viết bằng Go + [Ebitengine](https://ebitengine.org).
Tên và art đều là tạm; mọi hình vẽ là pixel art dựng bằng code trong `internal/client` (`art.go` cho sprite, `ground.go` cho mặt đất và nước, `pix.go` là bộ vẽ pixel).

## Chạy

Trên Windows (cần Go 1.25+, không cần cgo):

    go run ./cmd/game
    go run ./cmd/game -name Tien            # tên điền sẵn khi tạo nhân vật; "Chơi tiếp" vào bằng tên này (bỏ trống: người chơi trong bản lưu)
    go run ./cmd/game -seed 42 -size 512    # seed và cỡ thế giới cho "Trò chơi mới" (bỏ -size thì chọn trong menu)
    go run ./cmd/game -save D:\save.json     # dùng file lưu khác

Mở game sẽ vào menu chính: **Chơi tiếp** (nếu đã có bản lưu), **Trò chơi mới** (chọn cỡ thế giới: Vừa 256, Lớn 512, Rất lớn 1024 ô), **Thoát**.

Sau khi chọn cỡ thế giới là màn hình **Tạo nhân vật**: gõ tên (tối đa 16 chữ; bỏ trống thì tên là "Người chơi 1"), chọn nam hoặc nữ, kiểu tóc (Ngắn, Dựng, Bob, Dài, Đuôi ngựa, Búi cao), màu tóc (8 màu) và màu da (6 màu: Sáng, Trung bình, Bánh mật, Ngăm, Nâu, Nâu sẫm), hoặc bấm **Ngẫu nhiên**; nhân vật xem trước tự xoay (nhấp vào để xoay tiếp) cho thấy cả bốn hướng. Nữ mặc váy, nam mặc áo và quần; màu áo vẫn theo người chơi. Ngoại hình được lưu cùng người chơi nên **Chơi tiếp** giữ nguyên; bản lưu cũ vẫn là nhân vật nam tóc ngắn nâu như trước. Bản lưu nhận ra người chơi theo tên, nên **Chơi tiếp** vào lại đúng nhân vật đã tạo.

Bản lưu nằm ở `%AppData%\SinhTon\save.json`. Game tự lưu mỗi phút, khi về menu và khi
đóng cửa sổ; trong game nhấn **Esc** để tạm dừng (thời gian trong game đứng yên) và lưu ngay.

Build `.exe` cho Windows từ Linux:

    make build-windows      # ra dist/survival-game.exe

Chạy client trên Linux cần cgo và các gói X11/GL dev (xem tài liệu cài đặt của Ebitengine).

## Âm thanh

Mọi âm thanh cũng được tổng hợp bằng code, không có file âm thanh nào:

- `internal/client/sfx.go`: hiệu ứng (chặt, đập đá, nhặt đồ, ăn, lửa, bước chân theo loại đất…) và âm thanh môi trường lặp (sóng biển, lửa, gió, dế).
- `internal/client/music.go`: ba bản nhạc nền (menu, ban ngày, ban đêm) được soạn và "chơi" bằng code rồi lặp liền mạch; nhạc tự chuyển theo giờ trong game.
- `internal/client/audio.go`: bộ trộn phần mềm (âm lượng, pan trái/phải, độ cao) đưa ra một luồng Ebitengine.
- `internal/client/sound_events.go`: server chỉ gửi trạng thái, nên client so hai snapshot liên tiếp để biết khi nào phát tiếng.

Âm lượng nhạc và hiệu ứng chỉnh trong menu chính hoặc menu Esc (A/D hoặc phím mũi tên), lưu ở `%AppData%\SinhTon\settings.json`.

## Điều khiển

| Phím | Tác dụng |
| --- | --- |
| WASD / mũi tên | Di chuyển |
| Giữ Space | Nhặt cành cây, hái quả, cắt cỏ, đập đá, chặt cây (cần rìu), thêm củi vào lửa; bắt thỏ, đánh rắn, bắt giun, câu cá |
| 1–9, 0 / lăn chuột | Chọn ô 1–10 trên hàng ô nhanh; món ở ô đó là món đang cầm (người khác cũng thấy) |
| E | Ăn món đang cầm; nếu nó không ăn được thì ăn món tốt nhất trong túi (đồ nướng trước, đồ sống sau cùng) |
| F | Nướng 1 miếng Thịt sống hoặc Cá sống (phải đứng trong vùng sáng của lửa trại đang cháy) |
| Q | Làm Rìu (2 Cành cây, 1 Đá) |
| C | Dựng lửa trại (2 Gỗ, 2 Cỏ khô, 3 Đá) |
| B | Mở/đóng túi đồ: 20 ngăn túi + 10 ô nhanh, mỗi ô chứa một loại (tối đa 40, rìu 1). Nhấp một ô để nhấc lên, nhấp ô khác để đặt, gộp hoặc đổi chỗ (chuột phải để bỏ). Shift+nhấp chuyển nhanh giữa túi và ô nhanh; 1–0 đưa món đang trỏ vào ô nhanh đó; mũi tên/WASD + Enter cũng dùng được. Nhấp vào công thức để chế tạo |
| M | Mở/đóng bản đồ: chỉ hiện những nơi bạn đã đi qua (trong vòng khoảng 10 ô quanh nhân vật), phần còn lại phủ sương; bản đồ đã khám phá được lưu theo từng người chơi. Có vị trí của bạn (icon đầu nhân vật), người chơi gần đó, lửa trại đã thấy. Góc trên bên phải màn hình luôn có minimap: vùng quanh bạn, chỉ hiện chỗ đã đi qua |
| R | Hồi sinh khi đã chết |
| Esc | Tạm dừng: tiếp tục, lưu game, chỉnh âm lượng, về menu, thoát |
| F11 | Toàn màn hình |

Mới vào bạn chỉ có hai bàn tay: nhặt cành cây rơi dưới đất, đập một tảng đá (quanh chỗ xuất phát luôn có sẵn vài cành và một tảng đá), rồi nhấn Q làm Rìu. Có rìu mới chặt được cây lấy gỗ.

Một ngày trong game dài 8 phút: khoảng 4 phút rưỡi ban ngày, hơn 1 phút chạng vạng và gần 2 phút rưỡi
ban đêm. Ban đêm, đứng ngoài vùng sáng của lửa trại sẽ bị lạnh và mất máu. Từ cuối chạng vạng, đom đóm
bay lập lòe trên đồng cỏ, trong rừng và đầm lầy.

### Động vật

| Con | Ở đâu | Cách bắt | Được gì |
| --- | --- | --- | --- |
| Thỏ | Đồng cỏ, rừng, quanh hang thỏ | Chạy vụt đi khi bạn tới gần (nhanh hơn bạn) nhưng chỉ chạy được một đoạn rồi phải dừng thở: đuổi theo và nhấn Space lúc đó. Nếu hang ở gần và bạn không đứng chắn đường, thỏ chui tọt vào hang. Ban đêm thỏ ngủ ngoài hang, bắt dễ. | 2 Thịt sống |
| Rắn | Bãi đá, bãi cát, quanh hang rắn | Lao tới cắn khi bạn lại gần (mất 10 máu mỗi lần cắn). Chậm hơn bạn, nên cứ đi xa là thoát. Đánh 3 lần thì chết. Lúc nghỉ thì cuộn tròn; tối đến chui vào hang. | 1 Thịt sống |
| Giun | Đồng cỏ, rừng | Bò chậm, nhặt là được. | 1 Giun (mồi câu) |
| Cá | Vùng nước sát bờ | Cầm Cần câu, có Giun, đứng trên bờ và nhấn Space gần con cá để quăng câu; xem cách câu bên dưới. | 1 Cá sống |
| Cua | Bãi cát sát mép nước (cả hồ) | Thấy bạn trong vòng khoảng 2,5 ô là bò ra nước và lặn mất (chỉ còn bọt nước), chờ không ai ở gần mới bò lên. Chặn đường ra nước rồi nhấn Space để bắt; cua bị dồn vào chỗ không có nước sẽ giơ càng và kẹp (mất 4 máu) nếu bạn sát quá. | 1 Thịt cua |
| Cá sấu | Ao trong đầm lầy (chỉ có ở thế giới mở) | Thường rình dưới nước, chỉ lộ đôi mắt: lúc đó không đánh được. Ai lại gần mép nước trong khoảng 3 ô thì nó lao lên nhanh hơn bạn chạy và cắn rất đau (18 máu, giáp giảm bớt), rồi lặn lại và phải chờ vài giây mới phục kích tiếp. Trên cạn nó chậm hơn bạn (trong bùn thì chỉ hơi chậm hơn) và chỉ đuổi khoảng 3 giây. Thỉnh thoảng nó lên bờ phơi nắng: lúc đó mới đánh được, cần 6 nhát tay không (3 nhát giáo); bị thương thì nó bò về nước. | 3 Thịt sống |

Động vật bị bắt hết sẽ dần xuất hiện lại ở chỗ xa người chơi. Động vật không được lưu: mở lại bản lưu thì đảo có đàn mới.

**Hang thỏ** (ụ đất có miệng hang) nằm trên đồng cỏ và trong rừng, mỗi hang có vài con thỏ. Thỏ đang chạy trốn sẽ lao về hang nếu hang cách nó không quá 6 ô và bạn không đứng gần hang hơn nó; vào hang rồi thì không bắt được. Thỏ trốn ít nhất 8 giây và chỉ ló ra khi không còn ai đứng trong vòng 4 ô, nên muốn bắt thì chặn đường về hang, hoặc đứng xa chờ nó ra. **Hang rắn** (kẽ đá có xác da rắn lột) nằm ở bãi đá, bãi cát, không bao giờ gần chỗ xuất phát. Khi trời tối rắn bỏ cả con mồi để về hang và ở trong đó tới sáng, nên ban đêm không lo rắn cắn. Thỏ và rắn mới sinh ra chui ra từ hang. Hang không phá được và được lưu cùng bản đồ.

### Câu cá

1. Cầm Cần câu trên tay (chọn ở ô nhanh), có Giun trong túi, đứng trên bờ và nhấn Space gần con cá: phao rơi xuống chỗ con cá.
2. Đợi khoảng 1,5–4,5 giây. Khi cá cắn câu, phao chìm, có tiếng "tõm" và dấu **!**: nhấn Space trong khoảng 0,7 giây.

| Khi nào nhấn | Kết quả |
| --- | --- |
| Đúng lúc cá cắn | Được 1 Cá sống, tốn 1 Giun |
| Trước khi cá cắn | Cá sợ bơi đi, giữ được Giun |
| Không nhấn kịp | Cá ăn mất mồi: mất 1 Giun |

Giữ Space không tự giật được: phải nhấn lại đúng lúc. Đi chỗ khác hoặc cất cần câu thì thu dây (không mất Giun). Con cá đang mắc câu của người này thì người khác không câu được.

### Nướng thịt và cá

Đứng trong vùng sáng của lửa trại đang cháy rồi nhấn F (hoặc nhấp công thức trong túi đồ): mỗi lần nướng 1 miếng, thịt trước rồi đến cá.

| Món | No bụng | Máu |
| --- | --- | --- |
| Thịt sống | +15 | −2 |
| Cá sống | +10 | −1 |
| Thịt nướng | +35 | +5 |
| Cá nướng | +28 | +6 |
| Thịt cua | +10 | −1 |
| Cua nướng | +26 | +6 |

Đồ sống vẫn đỡ đói nhưng làm mất chút máu; ăn đồ sống không bao giờ làm bạn chết (máu thấp nhất còn 1).

## Thế giới

Trò chơi mới tạo một thế giới mở (`sim/openworld.go`): một lục địa lớn ở giữa, vành đảo nhỏ quanh bờ, và các vùng:

| Vùng | Có gì | Lưu ý |
| --- | --- | --- |
| Đồng cỏ, rừng | Cây, bụi quả, cỏ, cành, thỏ, giun | Chỗ xuất phát luôn ở đây |
| Bờ biển | Sóng vỗ vào bờ, bãi sò trên cát sát nước (nhặt tay, nướng thành Sò nướng) | Sò mọc lại mỗi ngày |
| Đồi đá | Tảng đá, đá cuội | Bao quanh các dãy núi |
| Núi | Vách đá cao, không leo được | **Cửa hang** là những vòm đen ở vách phía nam. Bước vào thì mái đá biến mất, cảnh tối lại (mang Đuốc), pha lê phát sáng trong vách, có tiếng nước nhỏ giọt và nhạc riêng; trong hang có tảng đá, mỏ than, quặng sắt, nấm |
| Hoang mạc | Xương rồng (cần rìu, ăn được), rắn | Ở phía nóng (phía nam) |
| Đầm lầy | Lau sậy (cho cỏ khô), nấm, ao có cá | Đi chậm trong bùn |
| Núi lửa | Tro, mỏ than, sông dung nham | Luôn ở xa chỗ xuất phát; đứng gần dung nham thì ấm, ban đêm dung nham phát sáng |

Than đào ở mỏ than (cần cúp đá); bỏ vào lửa trại cháy lâu gấp đôi gỗ. Bản lưu cũ vẫn giữ nguyên hòn đảo của nó.

Xem nhanh: `go run ./cmd/shot -open -size 512 -at cave` (hoặc desert, marsh, mountain, ash, lava); bản đồ toàn thế giới ra PNG: `OPEN_DUMP=<thư mục> go test ./internal/sim -run DumpOpen`.

## Chế tạo

Mở túi đồ (B) để xem mọi công thức và nhấp vào để làm. Món "cầm trên tay" phải nằm ở ô nhanh đang chọn (1–0).

| Món | Công thức | Tác dụng |
| --- | --- | --- |
| Rìu | 2 Cành cây, 1 Đá | Có trong túi thì chặt được cây, dỡ được tường (Q) |
| Cuốc chim | 2 Cành cây, 3 Đá | Có trong túi thì đập đá nhanh gấp đôi |
| Đuốc | 1 Cành cây, 1 Cỏ khô | Cầm trên tay: sáng và ấm quanh mình (cả người đứng gần), cháy 90 giây |
| Giáo | 2 Cành cây, 1 Đá | Cầm trên tay: đánh thú mạnh gấp đôi, với xa hơn nửa ô |
| Cần câu | 3 Cành cây, 2 Cỏ khô | Cầm trên tay mới câu được cá (vẫn tốn 1 Giun) |
| Lửa trại | 2 Gỗ, 2 Cỏ khô, 3 Đá | Ánh sáng và sưởi ấm ban đêm (C) |
| Tường gỗ | 2 Gỗ | Chặn đường người và thú; dỡ bằng rìu, trả lại 1 Gỗ |
| Giường cỏ | 6 Cỏ khô, 2 Gỗ | Nhấn Space lúc trời tối để ngủ: không bị lạnh, hồi máu nhanh; ai cũng ngủ thì đêm trôi nhanh gấp 8 |
| Thịt / Cá / Quả mọng nướng | 1 món sống | Đứng gần lửa trại, nhấn F |

## Kiến trúc

    cmd/game            client; chơi 1 người thì chạy luôn server trong cùng process
    internal/sim        luật chơi: sinh map, người chơi, tài nguyên, chế tạo, ngày/đêm
    internal/protocol   message giữa client và server
    internal/server     vòng tick 20Hz, kết nối, snapshot theo bán kính quan tâm
    internal/client     Ebitengine: input, vẽ, ánh sáng, HUD; nhân vật theo ngoại hình ở `look.go`, màn hình tạo nhân vật ở `charcreate.go`

- `sim` không biết gì về đồ hoạ hay mạng, chỉ tính theo tick, nên test được và chạy được trong server riêng.
- Server là nơi duy nhất quyết định trạng thái. Client gửi input, nhận snapshot và vẽ nội suy giữa hai snapshot gần nhất.
- Chơi 1 người vẫn đi qua đúng ranh giới client–server (`server.ConnectLocal`). Thêm co-op LAN là viết một transport TCP thoả `client.Conn`, không đụng tới luật chơi.

## Test

    make test    # sim, protocol, server (kèm race detector)
    make vet     # vet cả client cho Windows

Xem nhanh đồ hoạ mà không cần chơi (lưu một khung hình ra PNG):

    go run ./cmd/shot -o day.png
    go run ./cmd/shot -o night.png -time 0.85
    go run ./cmd/shot -o coast.png -coast
    go run ./cmd/shot -o animals.png -coast -animals          # mỗi loài một con cạnh người chơi
    go run ./cmd/shot -o fishing.png -coast -fishing bite -dir 0   # dây câu, phao lúc cá cắn
    go run ./cmd/shot -o menu.png -menu
    go run ./cmd/shot -o create.png -create                     # màn hình tạo nhân vật
    go run ./cmd/shot -o look.png -female -hair 4 -haircolor 4 -skin 2  # ngoại hình người chơi: tóc 0-5, màu tóc 0-7, màu da 0-5
    go run ./cmd/shot -o map.png -overlay map -frames 120   # đợi bản đồ vẽ xong
    go run ./cmd/shot -o bag.png -overlay bag

`SIM_DUMP=1 go test ./internal/sim -run Dump -v` in map ra ASCII để chỉnh thông số sinh map.
