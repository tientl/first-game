# Dino Isles: Survival

Game sinh tồn thời tiền sử 2D góc nhìn 3/4: săn và thuần hóa khủng long, làm đồ đá, mặc đồ da. Viết bằng Go + [Ebitengine](https://ebitengine.org).
Art là tạm; mọi hình vẽ là pixel art dựng bằng code trong `internal/client` (`art.go` cho sprite, `ground.go` cho mặt đất và nước, `pix.go` là bộ vẽ pixel). Riêng khủng long, thằn lằn bay và thằn lằn cổ dài vẽ từ hình trong `internal/client/dinos/`; các tư thế (bước, cúi ăn, gầm, vồ, vỗ cánh) suy ra từ hình đó trong `dinorig.go` và `dinorigs.go`.

## Chạy

Trên Windows (cần Go 1.25+, không cần cgo):

    go run ./cmd/game
    go run ./cmd/game -name Tien            # tên điền sẵn khi tạo nhân vật; "Chơi tiếp" vào bằng tên này (bỏ trống: người chơi trong bản lưu)
    go run ./cmd/game -seed 42 -size 512    # seed và cỡ thế giới cho "Trò chơi mới" (bỏ -size thì luôn là 1024, cỡ lớn nhất)
    go run ./cmd/game -save D:\save.json     # dùng file lưu khác

Mở game sẽ vào menu chính: **Chơi tiếp** (nếu đã có bản lưu), **Trò chơi mới** (thế giới luôn rộng nhất, 1024 × 1024 ô; chỉ chọn **độ khó**), **Thoát**.

**Độ khó** (`internal/sim/difficulty.go`), chọn khi tạo thế giới và lưu cùng bản lưu (bản lưu cũ là Cơ bản). Đất đai, thú và máu của thú như nhau ở mọi cấp; chỉ khác những gì thiên nhiên làm với người chơi, và cái giá của cái chết:

| | Thú đánh | Đói | Lạnh đêm đông | Bầy Troodon quanh mỗi người | Khi chết |
| --- | --- | --- | --- | --- | --- |
| Cơ bản | ×1 | ×1 | ×1 | 2 | giữ nguyên đồ |
| Thành thạo | ×1,5 | nhanh ×1,3 | ×1,5 | 3 | mất đồ, lấy lại ở xác |
| Chúa tể | ×2 | nhanh ×1,6 | ×2 | 4 | mất đồ, lấy lại ở xác |

**Xác người chơi** (`internal/sim/grave.go`): ở Thành thạo và Chúa tể, chết rồi hồi sinh thì về tay trắng; mọi thứ đang mang nằm lại trong xác ở chỗ chết. Xác vẽ đúng người đã chết (nam hay nữ, tóc, màu da, màu áo) nằm ngửa trên vệt máu, cạnh một túi da đựng đồ (`internal/client/bodyart.go`). Chưa quay lại lấy đồ mà chết thêm lần nữa thì xác cũ thành **bộ xương** (vẫn còn mảnh áo, túi đồ, vẫn lấy lại được), kể cả khi lần sau chết tay trắng. Xem nhanh: `go run ./cmd/shot -bodies`. Chết dưới nước hay chỗ có vật cản thì xác nằm ở ô đất trống gần nhất (trong 10 ô). Quay lại, nhấn Space vào xác để lấy lại đồ (túi đầy thì lấy được bao nhiêu hay bấy nhiêu); xác trống thì biến mất. Xác không mục, được lưu cùng game, ai cũng lấy được. Ở Cơ bản, hồi sinh là còn nguyên đồ.

Sau khi chọn độ khó là màn hình **Tạo nhân vật**: gõ tên (tối đa 16 chữ; bỏ trống thì tên là "Người chơi 1"), chọn nam hoặc nữ, kiểu tóc (Ngắn, Mào dựng, Bob tết, Dài, Bím dài, Búi cao), màu tóc (8 màu) và màu da (6 màu: Sáng, Trung bình, Bánh mật, Ngăm, Nâu, Nâu sẫm), hoặc bấm **Ngẫu nhiên**; nhân vật xem trước tự xoay (nhấp vào để xoay tiếp) cho thấy cả bốn hướng. Nhân vật theo kiểu **Viking**. Nam để râu quai nón cùng màu tóc, tết bím có hạt vàng; mặc áo chẽn thắt đai da, quần quấn xà cạp, ủng viền lông, áo choàng lông xám cài trâm tròn. Nữ mặc váy yếm Viking ngoài áo lót sáng màu, cài hai trâm bầu dục nối chuỗi hạt, cổ áo lông. Màu áo, váy theo người chơi. Ngoại hình được lưu cùng người chơi nên **Chơi tiếp** giữ nguyên; bản lưu cũ vẫn là nhân vật nam tóc ngắn nâu như trước. Bản lưu nhận ra người chơi theo tên, nên **Chơi tiếp** vào lại đúng nhân vật đã tạo.

Bản lưu nằm ở `%AppData%\SinhTon\save.json`. Game tự lưu mỗi phút, khi về menu và khi
đóng cửa sổ; trong game nhấn **Esc** để tạm dừng (thời gian trong game đứng yên) và lưu ngay.

Build `.exe` cho Windows từ Linux:

    make build-windows      # ra dist/survival-game.exe

Icon của game (khủng long cổ dài trên đảo dừa, cỡ 16 đến 256) vẽ bằng code trong `internal/client/gameicon.go`; game đặt nó cho cửa sổ và thanh tác vụ. Icon của file `.exe` trong Explorer nằm trong `cmd/game/rsrc_windows_amd64.syso`. Sửa icon xong thì chạy `go run ./cmd/icon -o <thư mục> -syso cmd/game/rsrc_windows_amd64.syso` để tạo lại file đó (kèm PNG từng cỡ, file `.ico` và ảnh xem trước trong thư mục).

Chạy client trên Linux cần cgo và các gói X11/GL dev (xem tài liệu cài đặt của Ebitengine).

## Chế độ dev

Để thử nhanh mọi tính năng (chỉ bật khi chạy với cờ `-dev`; chạy thường thì không có gì trong đây):

    go run ./cmd/game -dev                  # thế giới mới 256 × 256, seed cố định 20240601
    go run ./cmd/game -dev -seed 7 -size 512

- Bỏ qua menu, vào thẳng thế giới mới với nhân vật mặc định tên "Dev" (hoặc theo `-name`). Bản lưu riêng `dev-save.json` cạnh bản lưu thường (hoặc theo `-save`).
- Bất tử (không mất máu, không đói, không lạnh), hiện hết bản đồ ngay từ đầu, góc trên có nhãn **DEV**.
- Phím `` ` `` (hoặc **F12**) mở bảng lệnh ở dưới màn hình; ↑/↓ gọi lại lệnh cũ, Enter chạy, Esc hoặc `` ` `` đóng. Lệnh gửi tới server; server chỉ chạy khi được mở ở chế độ dev, ngoài ra thì từ chối.
- Tên đồ, tên thú, tên nơi gõ không dấu, không phân biệt hoa thường (`go` là Gỗ, `riu da lua` là Rìu đá lửa), gõ số thứ tự hay tên tiếng Anh thông dụng (`wood`, `rex`, `raptor`…) cũng được; tên có dấu cách thì để trong ngoặc kép hoặc gõ liền.

| Lệnh | Việc |
| --- | --- |
| `give <đồ> [số]` | cho đồ vào túi |
| `tp <làng\|trùm N\|hang\|kim tự tháp\|spawn\|x y>` | dịch chuyển tới làng, hang gần nhất, trùm N (1 Titanoboa … 6 Megalodon), kim tự tháp, chỗ hồi sinh hay toạ độ |
| `spawn <thú> [số]` | gọi thú ra quanh mình (trùm thì không) |
| `time <0-1\|day\|dusk\|night>` | vặn đồng hồ tới lúc đó (chỉ đi tới, không lùi) |
| `season <xuân\|hạ\|thu\|đông>` | sang đầu mùa đó |
| `weather <clear\|rain\|wind\|storm\|tornado\|snow\|auto>` | giữ trời theo ý (không lưu); `auto` trả lại tự nhiên |
| `speed <hệ số>` | tốc độ đi và bơi (0,1 tới 8) |
| `kill [bán kính]` | giết thú hoang trong 12 ô (thú nuôi thì không) |
| `heal`, `god [on\|off]`, `reveal`, `pos` | hồi đầy máu và no; bật/tắt bất tử; mở hết bản đồ; xem toạ độ |
| `list items [lọc]`, `list beasts [lọc]`, `help` | danh sách đồ, thú; các lệnh |

Mã ở `internal/sim/dev.go` (các việc), `internal/sim/devcmd.go` (đọc lệnh), `internal/server/dev.go`, `internal/client/devconsole.go`, `cmd/game/dev.go`. Bất tử, tốc độ và thời tiết ép không bao giờ vào bản lưu. Xem nhanh bảng lệnh: `go run ./cmd/shot -open -devconsole`.

## Âm thanh

Mọi âm thanh cũng được tổng hợp bằng code, không có file âm thanh nào:

- `internal/client/sfx.go`: hiệu ứng (chặt, đập đá, nhặt đồ, ăn, lửa, bước chân theo loại đất…) và âm thanh môi trường lặp (sóng biển, lửa, gió, dế).
- `internal/client/swimsound.go`: tiếng bơi. Nhảy xuống nước thì ùm một tiếng; bơi thì mỗi sải tay một tiếng rẽ nước, giữa hai sải có tiếng đạp chân; đứng nổi một chỗ thì nước óc ách quanh vai; leo lên bờ thì nước chảy ròng và nhỏ giọt. Dưới nước và trên thuyền không có tiếng bước chân.
- `internal/client/music.go`: ba bản nhạc nền (menu, ban ngày, ban đêm) được soạn và "chơi" bằng code rồi lặp liền mạch; nhạc tự chuyển theo giờ trong game.
- `internal/client/audio.go`: bộ trộn phần mềm (âm lượng, pan trái/phải, độ cao) đưa ra một luồng Ebitengine.
- `internal/client/sound_events.go`: server chỉ gửi trạng thái, nên client so hai snapshot liên tiếp để biết khi nào phát tiếng.
- `internal/client/roar.go`: tiếng gầm và tiếng kêu của thú. Mỗi loài một giọng: bạo chúa gầm (3 kiểu, có tiếng vọng; ở xa nghe trầm và đục, vọng tới khoảng 32 ô), cổ dài hú trầm dài, ba sừng, Torosaurus, gai lưng rống, voi ma mút rống vòi, raptor sủa từng tràng ngắn, Anzu "bùm" ba tiếng từ ngực, Quetzalcoatlus kêu khàn. Thú đứng hoặc đi thong thả thỉnh thoảng tự cất tiếng (mỗi lần chỉ một con, cách nhau ít nhất 2,5 giây); bạo chúa ngửa cổ há hàm khi gầm, ba sừng hất đầu, voi giơ vòi, Anzu xù mào. Bạo chúa còn gầm khi thấy người, khi lao đi săn, khi hạ được mồi và khi bị đánh; bầy raptor kêu khi hạ mồi.
- `internal/client/predsound.go`: giọng của thú săn mồi, làm cho đáng sợ. Tiếng gầm bạo chúa (`rexroar.go`) ghép nhiều con vật như tiếng gầm làm cho phim: gừ trầm trong ngực, tiếng hàm bật mở và hít hơi; rồi tiếng rống há miệng của mèo lớn từ ba cổ họng lệch tông nhau, một giọng trầm hơn bên dưới, tiếng rú kiểu voi rống ở đỉnh kèm một tiếng rú lệch tông nghe gai người, tiếng ầm hạ âm bên dưới; cuối cùng khò khè ướt trong cổ và phì hơi. Ép rè mạnh, vọng dài như trong thung lũng. Bạo chúa gầm trong vòng 14 ô thì màn hình rung theo, càng gần rung càng mạnh. Raptor rít khàn như ngỗng lẫn rắn khi lao vào; hổ răng kiếm gầm như mèo lớn khi nhắm mồi và gầm gừ khi vồ; cá sấu phì phò khi lao khỏi nước. Thú săn đang đuổi trong vòng 12 ô thì nghe tiếng nó tới gần: bạo chúa thở gừ, raptor gừ lách cách trong cổ, hổ gầm gừ.
- `internal/client/tribesound.go`: tiếng người của thổ dân, tổng hợp từ một giọng người (dây thanh rung, ba formant đổi theo nguyên âm, hơi thở), mỗi chiến binh một giọng cao thấp riêng. Thấy người lạ là hét xung trận (6 kiểu: "Ha-yaaa!", "Ooo-aaa!", "Hây-ya-haaa!", tiếng lưỡi "ay-ay-ay-ay" rền, "Hựh! Haaa!", "Yaaa!"; không bao giờ lặp lại kiểu vừa hét); tối đa 3 người đứng gần (trong 10 ô) lần lượt hô đáp lại ("Hô!", "Hai!", "Hup! Hup!"…) cách nhau chừng nửa giây. Một trận chỉ hét một lần: mất dấu rồi thấy lại trong 8 giây thì thôi. Đuổi theo thú để săn thì chỉ hô một tiếng ngắn. Đâm giáo thì gằn "hựh", bị đánh thì kêu đau, ngã xuống thì kêu một tiếng dài lịm dần.
- `internal/client/creaturevoice.go`: tiếng các loài còn thiếu, con càng to giọng càng trầm và dài. Gấu hang gầm gừ khi đi lang thang và gầm vang khi thấy người; nhện rít lách tách; bọ cạp kêu lách cách, chích thì vút một tiếng; khỉ "hu-hu-ha-ha" và thét lên khi hoảng; thỏ kêu chít chít khi bỏ chạy; Thescelosaurus kêu "quác" trầm qua mũi; thằn lằn biển Mosasaurus rền trầm dưới nước, nổi lên thì ùm và rền, thỉnh thoảng nghe tiếng rền từ đáy biển; Kraken rên ướt, ùng ục khi quấn thuyền. Thú bị đánh thì kêu đau (thú ăn cỏ lớn rống ngắn vỡ giọng, raptor kêu thét, chim và thằn lằn bay kêu khàn…), cả đàn ăn cỏ gần đó rống báo động một tiếng; thú chết thì rên một tiếng dài trầm dần. Tiếng nào cũng đặt đúng chỗ trong thế giới (xa thì nhỏ, lệch trái phải), và có giới hạn cho từng con lẫn từng loại tiếng để cả bầy không kêu loạn xạ. Các tiếng này tạo song song với các hiệu ứng khác nên gần như không làm chậm lúc mở game.

Âm lượng nhạc và hiệu ứng chỉnh trong menu chính hoặc menu Esc (A/D hoặc phím mũi tên), lưu ở `%AppData%\SinhTon\settings.json`.

## Điều khiển

Điều khiển chính bằng chuột, như các game sinh tồn trên PC: bàn phím để đi, chuột để nhắm và làm.

| Phím / chuột | Tác dụng |
| --- | --- |
| WASD / mũi tên | Di chuyển. Nhân vật luôn quay mặt theo con trỏ chuột (đi lùi thì bước chân chạy ngược); để yên chuột vài giây thì quay theo hướng đi như cũ. Lúc cưỡi thú hay đi thuyền thì vẫn lái như trước |
| Chuột trái (nhấp hoặc giữ) | Làm việc với thứ **dưới con trỏ**: chặt cây (cần rìu), đập đá, hái quả, cắt cỏ, nhặt cành, đánh hay bắt thú, thêm củi vào lửa, mổ xác, quăng câu và giật cần. Thứ sẽ bị tác động có khung trắng bao quanh; xa quá tầm với thì khung đỏ và chữ "Xa quá". Dưới con trỏ không có gì trong tầm thì làm thứ gần nhất (ưu tiên phía con trỏ). Nhấp vào ô trên hàng ô nhanh để chọn ô đó |
| Giữ Space | Như chuột trái, vào thứ gần nhất (chơi bằng bàn phím vẫn được) |
| Chuột phải | Dùng món đang cầm: đồ ăn, thuốc thì ăn; thịt sống, cá sống… khi đứng cạnh lửa trại đang cháy thì nướng; đuốc, đầu thú, hạt giống thì cắm, dựng, gieo vào ô dưới con trỏ (ô đó không được thì đặt trước mặt, về phía con trỏ); phân thì bón cho cây dưới con trỏ; thuyền thì thả xuống nước về phía con trỏ; cầm thức ăn của thú và trỏ vào nó thì cho nó ăn (thuần hóa). Món khác: không làm gì, có dòng nhắc |
| E | Tương tác: nhặt đống đồ dưới con trỏ (hoặc gần nhất), mở rương, lục xác, cưỡi thú đã thuần / xuống thú, lên thuyền / xuống thuyền. Lúc mở rương thì E đóng rương |
| C | Mở/đóng bảng chế tạo: mọi công thức, chia theo thẻ (Tất cả, Dụng cụ & vũ khí, Xây dựng, Nấu & nung, Đồ dùng; A/D hoặc mũi tên đổi thẻ, lăn chuột để cuộn). Món làm được ngay có viền xanh, món còn thiếu bị làm mờ, nguyên liệu thiếu hiện màu đỏ. Nhấp để làm; món xây dựng (lửa trại, tường, giường, rương, cửa…) thì nhấp rồi chọn chỗ đặt bằng chuột (xem "Xây dựng bằng chuột" bên dưới) |
| 1–9, 0 / lăn chuột | Chọn ô 1–10 trên hàng ô nhanh; món ở ô đó là món đang cầm (người khác cũng thấy) |
| Tab / I (hoặc B) | Mở/đóng túi đồ: 20 ngăn túi + 10 ô nhanh, mỗi ô chứa một loại (tối đa 40, rìu 1). Nhấp một ô để nhấc lên, nhấp ô khác để đặt, gộp hoặc đổi chỗ (chuột phải để bỏ). Shift+nhấp chuyển nhanh giữa túi và ô nhanh; 1–0 đưa món đang trỏ vào ô nhanh đó; mũi tên/WASD + Enter cũng dùng được; E ăn món đang trỏ. Nhấp vào một món để xem công thức liên quan, nhấp công thức để chế tạo. Vứt đồ: kéo một chồng ra ngoài khung túi rồi thả là vứt cả chồng xuống đất; chuột phải vào một ô (lúc không nhấc gì) hoặc G vứt 1 món ở ô đang trỏ, Shift+G cả chồng |
| M | Mở/đóng bản đồ: chỉ hiện những nơi bạn đã đi qua (trong vòng khoảng 10 ô quanh nhân vật), phần còn lại phủ sương; bản đồ đã khám phá được lưu theo từng người chơi. Có vị trí của bạn (icon đầu nhân vật), người chơi gần đó, lửa trại đã thấy. Góc trên bên phải màn hình luôn có minimap: vùng quanh bạn, chỉ hiện chỗ đã đi qua |
| R | Như E với thú cưỡi và thuyền (cưỡi / xuống, lên / xuống thuyền) |
| R, E hoặc Enter (khi đã chết) | Hồi sinh |
| V | Như chuột phải với đuốc, đầu thú, hạt giống, phân, thuyền (đặt vào ô dưới con trỏ hoặc trước mặt) — cho người quen dùng bàn phím |
| G | Vứt 1 món đang cầm xuống đất ngay trước mặt; Shift+G (hoặc Ctrl+G) vứt cả chồng. Xem "Vứt đồ ra khỏi túi" bên dưới |
| = / − (hoặc Ctrl + lăn chuột) | Thu phóng: nhìn gần hơn hoặc xa hơn; lưu trong settings.json. Khi mở bản đồ (M): lăn chuột hoặc = / − để phóng to bản đồ quanh chỗ bạn |
| Esc | Đang xây thì thôi xây; đang mở bảng nào thì đóng bảng đó; không thì tạm dừng: tiếp tục, lưu game, chỉnh âm lượng, về menu, thoát |
| F11 | Toàn màn hình |

Bảng nhỏ góc dưới bên trái màn hình luôn nhắc các phím, và dòng "Chuột phải" đổi theo món đang cầm (ví dụ "ăn quả mọng", "nướng thịt sống", "cắm đuốc xuống chỗ con trỏ").

**Xây dựng bằng chuột** (`internal/client/mouse.go`, `internal/sim/aim.go`): nhấp một món xây dựng trong bảng chế tạo (C) hoặc trong túi đồ thì bảng đóng lại và một **bóng mờ** của món đó đi theo con trỏ, nằm gọn trên ô dưới con trỏ: **xanh** là đặt được, **đỏ** là không (xa quá tầm với khoảng 2,5 ô, ô đã có thứ khác, nước, ô mình đang đứng, hay tường, hàng rào chắn chỗ người khác đang đứng). Nhấp chuột trái để dựng ở đó; còn nguyên liệu thì vẫn ở chế độ xây để dựng tiếp (tiện xây tường), hết nguyên liệu thì tự thôi. Chuột phải hoặc Esc để thôi xây. Cầm đuốc, đầu thú hay hạt giống thì bóng mờ cũng hiện khi con trỏ ở chỗ đất trống gần bạn (không trỏ vào thứ gì làm được): nhấp chuột trái hay chuột phải là cắm, dựng, gieo ở đó. Máy chủ luôn kiểm tra lại ô được chọn (tầm với, đất trống, không chắn người).

Nhắm bằng chuột cũng do máy chủ quyết: máy khách gửi chỗ con trỏ (tọa độ ô trong thế giới) kèm phím đang giữ, máy chủ chọn đúng thứ dưới con trỏ nếu trong tầm với của bạn (cùng luật tầm với như trước: giáo, cưỡi thú thì với xa hơn), không thì thứ gần nhất; người khác cũng thấy bạn vung đúng dụng cụ vào đúng thứ. Xem nhanh: `go run ./cmd/shot -overlay craft` (bảng chế tạo), `-build 0 -aim 1.5,1.2` (bóng mờ lửa trại; `-build 8` là tường gỗ), `-prey -aim 1.1,-0.4` (khung quanh con thú dưới con trỏ), `-aim 2,-0.2` (khung đỏ "Xa quá").

Mới vào bạn chỉ có hai bàn tay: nhặt cành cây rơi dưới đất, đập một tảng đá (quanh chỗ xuất phát luôn có sẵn vài cành và một tảng đá), rồi mở bảng chế tạo (C) làm Rìu. Có rìu mới chặt được cây lấy gỗ.

Một ngày trong game dài 16 phút: vào xuân phân và thu phân khoảng 9 phút ban ngày, 2 phút rưỡi chạng vạng và 5 phút ban đêm. Ngày dài dần tới giữa mùa hạ (đêm còn khoảng 4 phút) và ngắn dần tới giữa mùa đông (đêm gần 6 phút); thanh giờ góc trên phải đổi theo (`sim.DuskAt`, `sim.NightAt`). Ngủ qua đêm vẫn tốn chừng ấy no bụng như trước, nhưng màn đêm khi ngủ trôi lâu gấp đôi (khoảng 1 phút). Bản lưu cũ giữ đúng ngày và giờ trong ngày. Chỉ đêm **mùa đông** mới lạnh: đứng ngoài vùng sáng của lửa (lửa trại, lò nung, đuốc, dung nham) thì mất máu; đêm các mùa khác chỉ tối, không mất máu. Đứng trên tuyết không có áo lông, hoặc ra ngoài lúc mùa đông có tuyết hay gió mà không mặc ấm thì vẫn lạnh như trước. Từ cuối chạng vạng, đom đóm
bay lập lòe trên đồng cỏ, trong rừng và đầm lầy.

**Thú săn đêm: Troodon** (`internal/sim/nightstalker.go`). Khi trời tối hẳn, từng bầy 3 con Troodon (khủng long nhỏ, mắt rất to, săn đêm) mò ra từ bóng tối, cách người chơi 16–24 ô, mỗi người tối đa 2 bầy quanh mình. Chúng lặng lẽ áp sát từ nhiều phía, lao vào cắn (7 máu) rồi lùi ra, chạy chậm hơn bạn một chút. Chúng **sợ ánh sáng**: không bao giờ bước vào vùng sáng của lửa trại, lò nung, dung nham hay đuốc đang cháy, chỉ đứng ở rìa nhìn vào; bị ánh sáng chiếu tới thì chạy ra. Mắt chúng phát sáng trong đêm. Chúng không đụng tới người đang ngủ trên giường, đang bơi hay ngồi thuyền, và không xuống hang. Trời rạng sáng là bỏ đi mất. 50 máu (đấm tay gần như vô ích), cho 2 Thịt sống, 1 Da thú, 1 Xương. Nghe: tiếng gọi nhau lanh lảnh trong đêm, tiếng rít khi phát hiện bạn, tiếng lách cách trong cổ khi bầy tới gần.

### Động vật

| Con | Ở đâu | Cách bắt | Được gì |
| --- | --- | --- | --- |
| Thỏ | Đồng cỏ, rừng, quanh hang thỏ | Chạy vụt đi khi bạn tới gần (nhanh hơn bạn) nhưng chỉ chạy được một đoạn rồi phải dừng thở: đuổi theo và nhấp chuột vào nó (hoặc Space) lúc đó. Nếu hang ở gần và bạn không đứng chắn đường, thỏ chui tọt vào hang. Ban đêm thỏ ngủ ngoài hang, bắt dễ. | 2 Thịt sống |
| Rắn | Bãi đá, bãi cát, quanh hang rắn | Lao tới cắn khi bạn lại gần (mất 10 máu mỗi lần cắn). Chậm hơn bạn, nên cứ đi xa là thoát. Có 30 máu: tay không khoảng 3 đấm. Lúc nghỉ thì cuộn tròn; tối đến chui vào hang. | 1 Thịt sống |
| Giun | Đồng cỏ, rừng | Bò chậm, nhặt là được. | 1 Giun (mồi câu) |
| Cá | Vùng nước sát bờ | Cầm Cần câu, có Giun, đứng trên bờ và nhấp chuột vào con cá để quăng câu; xem cách câu bên dưới. | 1 Cá sống |
| Cua | Bãi cát sát mép nước (cả hồ) | Thấy bạn trong vòng khoảng 2,5 ô là bò ra nước và lặn mất (chỉ còn bọt nước), chờ không ai ở gần mới bò lên. Chặn đường ra nước rồi nhấp chuột vào nó để bắt; cua bị dồn vào chỗ không có nước sẽ giơ càng và kẹp (mất 4 máu) nếu bạn sát quá. | 1 Thịt cua |
| Cá sấu | Ao trong đầm lầy (chỉ có ở thế giới mở) | Thường rình dưới nước, chỉ lộ đôi mắt: lúc đó không đánh được. Ai lại gần mép nước trong khoảng 3 ô thì nó lao lên nhanh hơn bạn chạy và cắn rất đau (18 máu, giáp giảm bớt), rồi lặn lại và phải chờ vài giây mới phục kích tiếp. Trên cạn nó chậm hơn bạn (trong bùn thì chỉ hơi chậm hơn) và chỉ đuổi khoảng 3 giây. Thỉnh thoảng nó lên bờ phơi nắng: lúc đó mới đánh được, cần 6 nhát tay không (3 nhát giáo); bị thương thì nó bò về nước. | 3 Thịt sống |

### Máu và sát thương

Mỗi sinh vật có máu riêng (cột Máu trong các bảng; thỏ 10, rắn 30, raptor 120, gấu hang 450, voi ma mút 600, bạo chúa 2000). Mỗi đòn trừ máu bằng sát thương của thứ đang cầm trên tay, xê dịch ±20%; 10% số đòn là **chí mạng**, mạnh gấp 1,6 lần. Mỗi vũ khí vung nhanh chậm khác nhau. Thú nhỏ trúng đòn chỉ khựng lại tối đa một lần mỗi giây, nên đánh liên tục vẫn bị cắn trả. Bạo chúa cần ít nhất kiếm và giáp bạch kim. Con thú bị đánh hiện thanh máu trên đầu (xanh, vàng, rồi đỏ), số máu mất nổi lên mỗi đòn (đòn chí mạng màu cam, có dấu "!"). Trỏ vào con thú sẽ thấy máu còn lại và sát thương ước chừng của đòn bạn.

| Cầm trên tay | Sát thương mỗi đòn | Mỗi đòn cách nhau |
| --- | --- | --- |
| Tay không | 10 (khủng long, bò sát, thú lớn: chỉ 1) | 0,4 giây |
| Cuốc (đá → hắc diện thạch) | 9, 11, 13, 16, 19 | 0,7 giây |
| Cúp (đá → hắc diện thạch) | 12, 15, 19, 23, 28 | 0,7 giây |
| Rìu (đá → hắc diện thạch) | 14, 17, 21, 26, 32 | 0,7 giây |
| Giáo | 20 | 0,6 giây |
| Giáo ngà | 40 | 0,7 giây |
| Kiếm (đá lửa → hắc diện thạch) | 30, 42, 55, 75 | 0,5 giây |
| Rìu chiến | 70 | 0,9 giây |

**Rìu chiến** (2 Gỗ, 2 Ngọc bích mài, 1 Da thú; `battleaxe.go`): lưỡi kép bằng ngọc bích buộc dây da vào cán gỗ dài. Nhát chém nặng nhất trước dao hắc diện thạch, nhưng vung chậm nhất: giơ rìu ra sau đầu, bổ xuống, lưỡi cắm xuống đất một nhịp rồi mới kéo về. Chỉ để đánh, không chặt cây được. Mỗi người giữ một cái.

**Da dày**: cú đấm tay không chỉ có tác dụng với thỏ, rắn, cua, nhện, khỉ và thổ dân. Với mọi loài khủng long, cá sấu, bò sát biển, hổ răng kiếm, gấu, voi ma mút và bọ cạp, mỗi cú đấm chỉ trừ 1 máu, nên đấm tay gần như vô ích và con thú sẽ đánh trả. Muốn săn chúng thì cần ít nhất giáo hoặc rìu. Trỏ vào con thú khi tay không sẽ thấy ghi "da dày: đấm tay gần như vô ích". Cầm đồ không phải vũ khí (quả mọng, đá…) cũng tính là tay không.

**Đấm tay** (`punch.go`): tay không mà đánh thú thì nhân vật co tay rồi thụt nắm đấm về phía con thú, người hơi nhào theo, trúng đòn lóe tia trắng vàng ở nắm đấm. Hai tay đấm luân phiên (thẳng, rồi móc tay kia). Hái, nhặt, thêm củi thì tay chỉ vươn ra nhẹ, không lóe; đang cầm quả mọng, đá… thì món đó nằm trong nắm tay. Cưỡi thú vẫn đấm được từ trên lưng.

**Giật bắn người** (`jolt.go`): bị thú đánh, ong đốt hoặc sét đánh thì nhân vật giật nảy lên, bật ngược ra sau, hai tay hất ra, người lóe trắng rồi đỏ trong chừng một phần tư giây. Chỉ là hình, không đẩy nhân vật đi. Đói, lạnh, trúng độc, chết đuối thì không giật. Đi bộ, cưỡi thú, bơi hay trên thuyền đều giật. Xem nhanh: `go run ./cmd/shot -punch jab -age 4`, `-punch cross`, `-hurt -age 2`.

Bẫy, hàng rào chông, sét, cây ăn thịt, thú săn và thú nuôi cắn nhau cũng trừ theo cùng thang máu này.

### Khủng long và bò sát tiền sử

Chỉ có ở thế giới mở, không bao giờ ở gần chỗ xuất phát (loài nguy hiểm càng ở xa). Thú to bị đánh chỉ khựng lại một chút rồi làm tiếp việc đang làm. Thú biết đánh trả thì túi đầy vẫn đánh được; phần không vừa túi khi hạ nó sẽ mất.

| Con | Ở đâu | Tính nết | Máu | Được gì |
| --- | --- | --- | --- | --- |
| Khủng long ba sừng | Đồng cỏ, theo đàn 4 con | Hiền; đánh một con thì cả đàn quanh đó (8 ô) nổi giận khoảng 12 giây: lao tới húc (22 máu). Chạy chậm hơn bạn | 300 | 6 Thịt sống, 2 Da thú, 1 Xương |
| Khủng long gai lưng | Rừng, theo cặp | Hiền; bị đánh thì quất đuôi gai vào ai đứng gần (20 máu), đuổi rất chậm | 340 | 6 Thịt sống, 2 Da thú, 2 Xương |
| Khủng long cổ dài | Đồng cỏ, rừng, nhóm 3 con | Không bao giờ đánh; bị đánh thì lững thững bỏ đi | 700 | 12 Thịt sống, 4 Da thú, 4 Xương |
| Dakotaraptor | Rừng sâu, bầy 3 con | Thấy bạn trong 7 ô (ban đêm 10 ô) là đuổi, tới gần thì vồ cắn (9 máu) rồi lùi ra. Chạy chậm hơn bạn, nên cứ chạy là thoát; con bị thương nặng tự bỏ chạy | 120 | 2 Thịt sống, 1 Da thú, 1 Xương |
| Khủng long bạo chúa | Vùng tro quanh núi lửa, hoang mạc xa | Thấy bạn trong 8 ô thì gầm lên rồi lao tới cắn (35 máu). Chậm hơn bạn; đuổi tối đa 20 giây rồi bỏ | 2000 | 10 Thịt sống, 4 Da thú, 4 Xương, 2 Răng bạo chúa |
| Thằn lằn bay | Bay lượn trên đồi đá, bờ biển | Bay cao thì không đánh được; thấy bạn trong 6 ô thì bổ nhào mổ (8 máu) rồi bay lên. Đánh được khi nó bổ nhào hoặc đậu dưới đất | 40 | 2 Thịt sống, 1 Da thú |
| Thằn lằn cổ dài | Biển, cách bờ 2–6 ô | Bơi dưới nước (không đánh được), ai đứng trên bờ trong khoảng 4,5 ô thì ngoi lên vươn cổ cắn (16 máu). Lúc ngoi lên thì đánh vào cổ được từ trên bờ | 240 | 6 Thịt sống, 2 Xương, 1 Da thú |
| Voi ma mút | Vùng tuyết, đàn 3 con | Hiền; đánh một con thì cả đàn quanh đó nổi giận, lao tới húc ngà (26 máu). Chạy chậm hơn bạn | 600 | 10 Thịt sống, 4 Lông thú, 2 Ngà voi, 1 Da thú |
| Hổ răng kiếm | Vùng tuyết, đi một mình | Thấy bạn trong 9 ô thì rình lại gần, cách khoảng 3 ô thì vồ cắn (20 máu) rồi lùi ra vồ tiếp. Chạy chậm hơn bạn; còn 3 máu thì nằm gục 8 giây | 220 | 4 Thịt sống, 2 Lông thú, 1 Xương |
| Ornithomimus | Đồng cỏ, hoang mạc, đàn 5 con | Giống đà điểu, chạy rất nhanh (nhanh hơn bạn nhiều), thấy bạn trong 6 ô là chạy. Chạy 3 giây thì dừng lại thở dốc: lúc đó mới bắt được | 40 | 3 Thịt sống, 1 Da thú, 1 Xương |
| Thescelosaurus | Rừng, đồng cỏ, đàn 3 con | Ăn cỏ cỡ vừa, hiền; bị đánh thì bỏ chạy | 80 | 4 Thịt sống, 1 Da thú, 1 Xương |
| Anzu | Rừng, đầm lầy, theo cặp | Như con gà khổng lồ có mào. Ai lại gần trong 3,5 ô thì xù mào dọa, lại gần nữa thì đá (10 máu); không đuổi xa | 120 | 3 Thịt sống, 1 Xương |
| Torosaurus | Đồng cỏ, rừng, đàn 3 con | Họ hàng ba sừng, to hơn, diềm cổ lớn có lỗ. Đánh một con thì cả đàn húc (26 máu) | 380 | 8 Thịt sống, 3 Da thú, 2 Xương |
| Acheroraptor | Rừng sâu, bầy 5 con | Raptor nhỏ hơn Dakotaraptor, bầy đông hơn: vồ cắn 6 máu; chạy chậm hơn bạn | 80 | 1 Thịt sống, 1 Da thú |
| Quetzalcoatlus | Đồng cỏ, bãi cát, đồi đá, hoang mạc, đi một mình | Thằn lằn bay khổng lồ, đứng cao ngang hươu cao cổ. Đi bộ rình, thấy bạn trong 6 ô thì lại gần mổ (18 máu); còn 40% máu thì bay vụt đi chỗ khác | 250 | 6 Thịt sống, 2 Da thú, 3 Xương |
| Brachychampsa | Ao trong đầm lầy | Giống cá sấu mõm tù: rình dưới nước, ai lại gần mép ao 2,8 ô thì lao lên cắn (16 máu) | 120 | 3 Thịt sống, 1 Da thú |
| Champsosaurus | Ao trong đầm lầy | Giống cá sấu mõm dài mảnh: rình dưới nước, ai lại gần 2,2 ô thì đớp (10 máu) | 90 | 2 Thịt sống, 1 Da thú |

**Ăn, uống và săn mồi.** Thú hoang cũng biết đói và khát:

- **Thú ăn cỏ** (ba sừng, Torosaurus, gai lưng, cổ dài, voi ma mút, Thescelosaurus, Ornithomimus, Anzu) cứ khoảng 40–90 giây lại cúi xuống gặm cỏ ngay chỗ đang đứng, chừng 6 giây, miệng nhai (voi ma mút cuộn vòi nhổ cỏ đưa lên miệng). Ai lại gần trong 2,5 ô là chúng thôi ăn; Thescelosaurus và Ornithomimus nhát hơn, chỉ cần bạn cách 6 ô là đã ngẩng lên.
- **Uống nước**: cứ vài phút một lần, con thú khát đi tìm mép nước gần nhất (trong khoảng 22 ô), cúi xuống uống khoảng 4 giây. Mép ao đầm lầy là chỗ nguy hiểm: cá sấu, Brachychampsa và Champsosaurus đang rình sẽ lao lên đớp con đang uống, và con kia bỏ chạy.
- **Thú săn mồi** đói thì đi tìm con mồi vừa sức trong khoảng 11 ô, đuổi theo, cắn tới khi con mồi gục, rồi đứng tại chỗ xé thịt ăn (8–12 giây, hàm há ra ngậm vào). Bạo chúa săn Thescelosaurus, Ornithomimus, ba sừng, Torosaurus và cổ dài; Dakotaraptor, Acheroraptor và hổ răng kiếm săn thỏ, Ornithomimus và Thescelosaurus; Quetzalcoatlus mổ thỏ và cua. Con mồi chạy tới khi hết hơi (khoảng 4 giây) thì phải dừng thở. Ba sừng, Torosaurus, gai lưng và voi ma mút thì không chạy mà quay lại húc kẻ săn; kẻ săn mất nửa máu sẽ bỏ cuộc.
- **Người chơi vẫn là ưu tiên**: thú săn đang đuổi mồi mà thấy bạn ở gần thì bỏ mồi quay sang bạn như trước. Đứng xa mà xem thì an toàn hơn.
- Bảng tên trên đầu con thú ghi "(đang ăn)", "(đang uống)" hoặc "(đang săn)". Thú đã thuần hóa thì không tự đi săn hay đi uống: chủ cho ăn.

Động vật bị bắt hết sẽ dần xuất hiện lại ở chỗ xa người chơi. Động vật không được lưu: mở lại bản lưu thì đảo có đàn mới.

**Hang thỏ** (ụ đất có miệng hang) được đào trên đồng cỏ gần bìa rừng (thỏ cần chỗ nấp), mỗi hang một bầy 2–3 con, các bầy ở cách xa nhau. Thỏ đang chạy trốn sẽ lao về hang nếu hang cách nó không quá 6 ô và bạn không đứng gần hang hơn nó; vào hang rồi thì không bắt được. Thỏ trốn ít nhất 8 giây và chỉ ló ra khi không còn ai đứng trong vòng 4 ô, nên muốn bắt thì chặn đường về hang, hoặc đứng xa chờ nó ra. **Hang rắn** (kẽ đá có xác da rắn lột) chủ yếu ở bãi đá, thưa thớt ở bãi cát, sa mạc và tro núi lửa, không bao giờ gần chỗ xuất phát. Rắn sống đơn độc: mỗi hang một con, mỗi con giữ một vùng rộng. Khi trời tối rắn bỏ cả con mồi để về hang và ở trong đó tới sáng, nên ban đêm không lo rắn cắn. Thỏ và rắn mới sinh ra chui ra từ hang còn chỗ: thỏ sinh nhanh (khoảng 30 giây một con), rắn rất chậm (khoảng 2 phút một con). Hang không phá được và được lưu cùng bản đồ.

### Câu cá

1. Cầm Cần câu trên tay (chọn ở ô nhanh), có Giun trong túi, đứng trên bờ và nhấp chuột vào con cá (hoặc Space khi đứng gần nó): phao rơi xuống chỗ con cá.
2. Đợi khoảng 1,5–4,5 giây. Khi cá cắn câu, phao chìm, có tiếng "tõm" và dấu **!**: nhấp chuột (hoặc Space) trong khoảng 0,7 giây.

| Khi nào nhấn | Kết quả |
| --- | --- |
| Đúng lúc cá cắn | Được 1 Cá sống, tốn 1 Giun |
| Trước khi cá cắn | Cá sợ bơi đi, giữ được Giun |
| Không nhấn kịp | Cá ăn mất mồi: mất 1 Giun |

Giữ chuột hay Space không tự giật được: phải nhấn lại đúng lúc. Đi chỗ khác hoặc cất cần câu thì thu dây (không mất Giun). Con cá đang mắc câu của người này thì người khác không câu được.

### Nướng thịt và cá

Đứng trong vùng sáng của lửa trại đang cháy , cầm món sống trên tay (thịt, cá, cua, sò, khoai…) rồi nhấp chuột phải; hoặc nhấp công thức trong bảng chế tạo (C) hay trong túi đồ. Mỗi lần nướng 1 miếng.

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
| Đồng cỏ, rừng rậm | Cây, bụi quả, cỏ, cành, thỏ, giun. Rừng mọc dày, nền phủ dương xỉ, lá to và cây bụi dưới bóng tán lá; đồng cỏ có cỏ xanh và hoa dại. Thế giới mới có rừng rộng gấp đôi | Chỗ xuất phát luôn ở đây |
| Bờ biển | Sóng vỗ vào bờ, bãi sò trên cát sát nước (nhặt tay, nướng thành Sò nướng) | Sò mọc lại mỗi ngày |
| Đồi đá | Tảng đá, đá cuội | Bao quanh các dãy núi |
| Núi | Vách đá cao, không leo được | **Cửa hang** là những vòm đen ở vách phía nam. Bước vào thì mái đá biến mất, cảnh tối lại (mang Đuốc), pha lê phát sáng trong vách, có tiếng nước nhỏ giọt và nhạc riêng; trong hang có tảng đá, mỏ than, mạch ngọc bích, nấm |
| Hoang mạc | Xương rồng (cần rìu, ăn được), rắn | Ở phía nóng (phía nam) |
| Đầm lầy | Lau sậy (cho cỏ khô), nấm (thường mọc trên khúc gỗ mục), ao có cá. Bùn xanh rêu lỗ chỗ vũng nước; ao nước xanh đục viền bùn, có lá súng, rong; cây bụt mọc rủ rêu và thân cây chết (chặt vẫn ra gỗ) | Đi chậm trong bùn |
| Núi lửa | Tro, mỏ than, sông dung nham | Luôn ở xa chỗ xuất phát; đứng gần dung nham thì ấm, ban đêm dung nham phát sáng |
| Vùng tuyết | Thông phủ tuyết, đá, cành cây, voi ma mút, hổ răng kiếm | Phía bắc lạnh nhất (chỉ có ở thế giới tạo mới). Đứng trên tuyết là lạnh cóng cả ngày lẫn đêm, trừ khi có Áo lông trong túi, đang cưỡi voi ma mút hoặc đứng gần lửa, đuốc. Đi trong tuyết chậm hơn, tuyết rơi lất phất |

Than đào ở mỏ than (cần cúp đá); bỏ vào lửa trại cháy lâu gấp đôi gỗ. Bản lưu cũ vẫn giữ nguyên hòn đảo của nó.

**Cây ăn quả** (`sim/fruit.go`) mọc ở mọi thế giới: cây sung (rừng, đồng cỏ), cây bạch quả (đồng cỏ, ít hơn trong rừng), cây dừa (bãi cát), cây chuối (rừng, đầm lầy). Nhấp chuột vào cây đang có quả để hái bằng tay (3 Quả sung, 4 Hạt bạch quả, 2 Dừa, 3 Chuối); quả ra lại sau 1 ngày (bạch quả 1,5 ngày, dừa 2 ngày). Cây đã hết quả thì chặt được bằng rìu như cây thường (không có rìu thì báo cần rìu): cây sung, cây bạch quả 4 nhát rìu đá, đổ cho 4 Gỗ; cây dừa 3 nhát, 3 Gỗ; cây chuối 2 nhát, 2 Gỗ; hay rơi thêm Cành cây. Cây đang có quả thì Space luôn hái quả trước. Giữ Space để hái thì không chặt luôn: muốn chặt, thả Space ra rồi nhấn lại. Cây đổ để lại gốc, 3 ngày sau gốc mọc lại đúng loại cây đó, lúc đầu chưa có quả, rồi ra quả như thường.

Xem nhanh: `go run ./cmd/shot -open -size 512 -at cave` (hoặc desert, marsh, mountain, ash, lava); bản đồ toàn thế giới ra PNG: `OPEN_DUMP=<thư mục> go test ./internal/sim -run DumpOpen`.

### Thuần hóa và cưỡi

Cầm thức ăn nó thích trên tay (ô nhanh đang chọn), trỏ chuột vào con thú rồi nhấp **chuột phải** (mỗi giây một lần); nhấp chuột trái hay Space vào nó lúc cầm thức ăn của nó cũng là cho ăn, không phải đánh. Nhãn trên đầu con thú ghi số bữa đã ăn. Đánh nó thì công thuần hóa mất hết. Thú đã thuần là của bạn mãi, được lưu cùng bản lưu, có trái tim đỏ trên đầu (thú của người khác có tim trắng, không đánh và không cưỡi được).

| Con | Cho ăn | Cưỡi được | Ghi chú |
| --- | --- | --- | --- |
| Khủng long ba sừng | 5 Quả mọng, lúc nó đang hiền | Có, chạy 5,5 ô/giây | Đi theo và húc thú dữ đang tấn công bạn |
| Khủng long gai lưng | 5 Quả mọng, lúc nó đang hiền | Không | Đi chậm, quất đuôi gai vào thú dữ quanh bạn |
| Dakotaraptor | 3 Thịt sống, chỉ khi nó đã bị đánh gục (còn 2 máu thì nó nằm rạp 8 giây rồi mới bỏ chạy) | Có, chạy 6,5 ô/giây | cắn thú dữ quanh bạn |
| Voi ma mút | 6 Cỏ khô, lúc nó đang hiền | Có, chạy 5 ô/giây | Cưỡi voi thì không bị lạnh trên tuyết; húc thú dữ quanh bạn |
| Hổ răng kiếm | 4 Thịt sống, chỉ khi nó đã bị đánh gục | Không | Cắn thú dữ quanh bạn |
| Ornithomimus | 4 Quả mọng, lúc nó đang dừng thở (cho ăn xong nó đứng yên vài giây) | Có, chạy 7,5 ô/giây — nhanh nhất | Không đánh |
| Torosaurus | 6 Quả mọng, lúc nó đang hiền | Có, chạy 5 ô/giây | Húc thú dữ quanh bạn |
| Anzu | 5 Quả mọng, lúc nó đang hiền | Không | Đá thú dữ quanh bạn |
| Thescelosaurus | 4 Quả mọng | Không | Không đánh |
| Acheroraptor | 2 Thịt sống, khi nó đã bị đánh gục | Không | Cắn thú dữ quanh bạn |

Thú cưng đi theo bạn (bỏ xa quá 24 ô thì nó tự tìm tới), tự hồi máu dần, và lao vào bất kỳ thú dữ nào đang đuổi hay cắn bạn trong vòng 6 ô; con nó hạ được thì bạn nhận chiến lợi phẩm. Nhấp chuột trái (hoặc Space) vào thú của mình để bảo nó đứng chờ hoặc gọi đi theo; cầm thức ăn của nó thì chuột phải là cho ăn hồi máu. Có Yên da trong túi, đứng gần thú cưỡi được của mình rồi nhấn **E** để cưỡi, nhấn **E** lần nữa để xuống. Khi cưỡi, bạn với xa hơn nửa ô; chết, ngủ hoặc thoát game thì tự xuống thú. **Yên da**: 4 Da thú, 2 Cỏ khô, 1 Xương. **Áo lông**: 4 Lông thú, 2 Da thú, giữ ấm trên tuyết. **Giáo ngà**: 2 Cành cây, 1 Ngà voi, 1 Cỏ khô; đánh mạnh gấp bốn tay không, với xa nhất.

## Quần đảo và biển khơi (thế giới tạo mới)

Từ bản này, trò chơi mới tạo một **quần đảo**: đảo nhà ở giữa là đồng cỏ hiền lành, xung quanh là biển rộng. Mỗi đảo là một vùng đất, đảo càng xa càng khắc nghiệt: rừng sâu và đồi núi, rồi đầm lầy, rồi núi tuyết và hoang mạc, ngoài rìa là đảo núi lửa. Muốn đi đâu cũng phải đóng thuyền.

| Món | Công thức | Tác dụng |
| --- | --- | --- |
| Bè gỗ | 8 Gỗ, 4 Cỏ khô | Chèo đi mọi hướng như nhau, chậm (3 ô/giây), chịu 60 sát thương |
| Thuyền buồm | 20 Gỗ, 4 Da thú, 8 Cỏ khô | Xuôi gió rất nhanh (tới 7 ô/giây), ngược gió thì bò; chịu 140 sát thương |

- Cầm thuyền trên tay, đứng sát mép nước quay mặt ra biển, trỏ chuột ra nước rồi nhấp **chuột phải** để thả. **E** để lên thuyền, chèo vào sát bờ rồi nhấn **E** để lên bờ. WASD để lái. Khi ở trên thuyền, góc trái có thanh độ bền thuyền và mũi tên chỉ hướng gió; gió đổi theo thời tiết.
- **Bơi**: ở thế giới quần đảo ai cũng bơi được, nhưng chậm và chỉ được khoảng 8 giây, sau đó kiệt sức và chết đuối dần. Khi bơi không làm gì được bằng tay.
- **Quái vật biển** sống ở biển sâu, cách bờ xa (gần bờ thì an toàn, chỉ có thằn lằn cổ dài):
  - **Thương long** (Mosasaurus) bơi ngầm, chỉ thấy bóng đen và sống lưng, nhanh hơn bè nhưng chậm hơn thuyền buồm xuôi gió. Tới gần thì ngoi lên đớp: thuyền mất 30 độ bền, người trên thuyền mất 8 máu, người đang bơi mất 34 máu. Sau cú đớp nó ở trên mặt nước vài giây: lúc đó mới đánh được, rồi lặn xuống.
  - **Mực khổng lồ** nằm sâu hơn nữa. Thuyền đi ngang qua là bị xúc tu giữ chặt 7 giây, mỗi giây thuyền mất 8 độ bền. Đánh nó mất 12 máu thì nó buông ra.
  - Hạ quái vật biển ngoài khơi thì đồ nổi trên mặt nước, chèo lại nhặt.
- **Kim tự tháp** đứng giữa đảo sa mạc lớn nhất (không bao giờ là đảo nhà): khối đá 7×5 ô, trong cửa có rương cổ giữ kho báu quý nhất. **Bọ cạp khổng lồ** sống trên cát sa mạc, vài con canh quanh kim tự tháp: chậm hơn bạn, kẹp bằng càng (10 máu) rồi chích đuôi (14 máu, trúng độc 8 giây). Xem nhanh: `go run ./cmd/shot -open -at pyramid -zoom -1`.
- **Hang ngầm**: quần đảo không có núi. Đảo nào cũng có **cửa hang** (trừ các bãi cát nhỏ dưới 60 ô; đảo lớn có 2 đến 3 cửa; trên đảo nhà cửa hang cách chỗ xuất phát ít nhất 16 ô, và không bao giờ nằm giữa làng). Cửa hang là một vòm đá nằm trên mặt đất; đi vào là xuống ngay mạng hang tối dưới lòng đất (nhện, gấu, than, quặng, nấm phát sáng, rương cổ). Chỗ xuống là cửa hang nhìn từ bên trong, có dốc đất dẫn lên chỗ nắng chiếu vào: đi lên đó là ra lại đúng cửa hang. Đang giữ phím đi thì không bị kéo ngược lại; thả phím ra rồi mới đi qua được lần nữa. Bản đồ (M) hiện tầng bạn đang ở: mặt đất hoặc hang. Xem nhanh: `go run ./cmd/shot -open -at cavemouth` và `-at cavein -frames 90`.
- Thuyền đậu lại được lưu cùng bản lưu; lưu game khi đang ở trên thuyền thì lúc vào lại vẫn ở trên thuyền.

Xem nhanh: `go run ./cmd/shot -open -sea`; bản đồ toàn thế giới: `OPEN_DUMP=<thư mục> go test ./internal/sim -run DumpOpen`.

## Thời bộ lạc (thế giới tạo mới)

Thế giới mới (kiểu `StyleTribal`) có thêm những thứ dưới đây; bản lưu cũ giữ nguyên địa hình cũ nhưng vẫn có mùa, thời tiết, đồ rơi, xác thú, đồ xây mới.

- **Đồ rơi dưới đất**: chặt cây, đập đá, mổ xác, mở rương cổ thì đồ văng ra đất. Nhấn E (đống dưới con trỏ, không thì đống gần nhất), nhấp chuột vào đống, giữ Space (đống gần nhất trước tiên) hoặc đi ngang qua để nhặt. Chặt cây hay rơi thêm Cành cây, đôi khi Hạt cây; đập đá đôi khi rơi Đá cuội. Đồ nằm đất 5 phút thì mục mất.
- **Vứt đồ ra khỏi túi** (mọi thế giới; `internal/sim/toss.go`, `internal/client/toss.go`): G vứt 1 món đang cầm, Shift+G (hoặc Ctrl+G) cả chồng; trong túi đồ (Tab) thì kéo một chồng ra ngoài khung túi rồi thả, chuột phải vào một ô, hoặc G / Shift+G với ô đang trỏ. Đồ rơi thành một đống ngay trước mặt (cách chừng 1 ô, theo hướng đang nhìn); phía trước là tường hay nước thì rơi ngay dưới chân. Vứt tiếp cùng loại vào chỗ đó thì dồn vào đống cũ (tối đa một ô túi mỗi đống). Đống vứt ra là đồ rơi như mọi đồ rơi khác: ai cũng nhặt được bằng Space hoặc đi ngang qua, được lưu cùng thế giới, 5 phút thì mục (trừ đồ huyền thoại). Riêng người vứt, đang đứng trên đống đó thì không tự nhặt lại khi đứng yên; bước ra rồi đi vào lại mới nhặt (Space thì nhặt ngay). Đang bơi hay ngồi thuyền thì chỉ vứt được lên bờ trước mặt; ngoài biển thì không vứt được (hiện thông báo, đồ vẫn trong túi) để khỏi mất đồ. Đã chết, đang ngủ hay đang mở rương thì không vứt được. Vứt cây đuốc đang cháy trên tay thì đuốc tắt. Có tiếng "bịch" nhẹ khi vứt.
- **Xác thú**: hạ thú thì xác nằm lại trên vũng máu (thú bốn chân lật ngửa). Nhấp chuột vào xác để mổ (cầm dao thì nhanh gấp đôi): ra thịt, da, xương, **đầu thú**, và **Da khủng long** nếu là khủng long. Xác để 4 phút thì rữa mất. Xác thổ dân thì lục soát (có khi được giáo).
- **Đầu thú**: cầm đầu trên tay, nhấp chuột phải (hoặc chuột trái vào chỗ đất trống) để cắm lên cọc ở ô dưới con trỏ làm vật trang trí; nhấp chuột trái vào nó để gỡ xuống.
- **Mùa và thời tiết**: mỗi mùa 12 ngày (xuân, hạ, thu, đông), một năm 48 ngày. Thời tiết đổi vài phút một lần: mưa (lửa cháy nhanh gấp đôi), gió, giông bão (sét đánh gần người: 35 máu, cây bị sét thì đổ), lốc xoáy (cuốn người quay vòng, mất máu, quật đổ cây), tuyết. Mùa đông khi có tuyết hay gió thì ở ngoài trời rất lạnh kể cả ban ngày (mặc Áo lông, Áo da thú hoặc Giáp da khủng long, hoặc đứng gần lửa, đuốc); mặt đất phủ tuyết; ao hồ (không phải biển) đóng băng, đi lên được, cá trốn dưới băng. Mùa thu lá cây ngả vàng.
- **Làng thổ dân**: lều gỗ mái tranh quanh cột vật tổ và đống lửa không bao giờ tắt. Ở quần đảo, đảo nào cũng có làng, trên mọi loại đất (rừng, đồng cỏ, sa mạc, tuyết, đầm lầy, tro núi lửa). Đảo lớn có 2 đến 3 làng mỗi làng 5 thổ dân; đảo nhỏ dưới 600 ô có làng 3 thổ dân; đảo nhỏ hoặc hẹp quá thì chỉ có xóm nhỏ 3 lều, 2 thổ dân; bãi cát dưới 90 ô thì không có. Trên đảo nhà, làng cách chỗ xuất phát ít nhất 22 ô. Làng luôn tránh chỗ dựng kim tự tháp. (Thế giới kiểu lục địa cũ vẫn như trước: vài làng xa chỗ xuất phát.) Thổ dân da đen, mặt vẽ sơn trắng, không mặc áo, cầm giáo: thấy bạn trong 7 ô (đêm 5 ô) là xông tới đâm (12 máu), bị thương nặng thì chạy về làng. Họ cũng đi săn thỏ, Thescelosaurus, Ornithomimus. Làng còn lều thì cứ một phút gửi thêm người.
- **Đường mòn**: nối chỗ xuất phát với các làng và cửa hang, giữa các làng với nhau; đi trên đường nhanh hơn. Rừng thưa hơn trước.
- **Hang sâu**: núi nhiều hơn, hang có nhiều nhánh và hầm rộng, ăn sâu vào lòng núi. Trong hang có **nhện độc** (nhả tơ làm bạn đi chậm, cắn có độc mất máu dần) và **gấu hang** (cào rồi cắn). Chỗ sâu nhất có **rương cổ**: nhấp chuột hoặc E để mở, càng sâu đồ càng quý (bình máu, quặng, hắc diện thạch, răng bạo chúa, hạt giống...).
- **Hoa và hạt**: 6 loại hoa (anh túc đỏ, cúc vàng, hoa lam, oải hương, cúc trắng, hồng dại), hái được hoa và hay được Hạt hoa. Hạt quả mọng (hái bụi quả), Hạt cây (chặt cây), Hạt phát sáng, Hạt cây ăn thịt: cầm hạt, nhấp chuột phải để gieo ở ô dưới con trỏ (đất mềm).
- **Cây phát sáng**: nấm, dương xỉ và cây cao phát sáng xanh lam, tím vào ban đêm, trong rừng sâu, đầm lầy và hang. Hái được Bào tử phát sáng.
- **Cây ăn thịt người**: cây khổng lồ đầu đỏ ở đầm lầy và rừng xa, với tới khoảng 2 ô. Thú nhỏ và khủng long nhỏ (thỏ, rắn, cua, Thescelosaurus, Ornithomimus, raptor, Anzu, nhện, bọ cạp) lại gần là bị nuốt chửng, rồi cây ngậm miệng nhai khoảng 8 giây. Thú to bị cắn, người bị đớp mất 16 máu. Chặt bằng vũ khí hoặc rìu (10 nhát tay), rơi hạt; cây trồng từ hạt thì chỉ đớp thú, canh nhà cho bạn.
- **Rừng** (thế giới tạo mới): **cây cổ thụ khổng lồ** thỉnh thoảng mọc giữa rừng, gốc rộng 3×2 ô không đi xuyên qua được. Phải chặt rất lâu (24 nhát rìu đá), đổ xuống cho 30 Gỗ, 8 Cành cây, 3 Hạt cây; gốc để lại mãi. **Cây chuối** ở rừng và đầm lầy, hái buồng bằng tay (3 Chuối: +14 no, +3 máu), mọc lại sau 1 ngày; hết buồng thì chặt được như mọi cây ăn quả. **Tổ ong** treo trên một số cây, cây cổ thụ hay có nhất. Space vào cây có tổ để lấy 3 Mật ong (+10 no, +15 máu), tổ đầy lại sau 1 ngày. Ong bay ra đốt (3 máu mỗi giây trong 8 giây) trừ khi bạn **cầm đuốc** (khói làm ong hiền). Chạy xa 10 ô hoặc nhảy xuống nước thì ong thôi đuổi; chặt cây có tổ cũng bị ong đốt. **Khỉ** sống theo đàn 4 con trong rừng, thấy người là chạy, chạy mệt thì ngồi thở. Lúc đó cầm Chuối cho ăn 3 lần là thuần, khỉ sẽ đi theo bạn. Xem nhanh: `go run ./cmd/shot -woods`.
- **Phân thú** (mọi thế giới, cả bản lưu cũ): thú ăn cỏ thỉnh thoảng thải phân sau lưng. Cổ dài để 3 cục; ba sừng, Torosaurus, gai lưng, voi ma mút và bạo chúa để 2; Thescelosaurus và Ornithomimus để 1. Trung bình mỗi con 1/3 ngày một lần, thú đã thuần thì nhanh gấp đôi. Đống phân có ruồi bay quanh, nằm được 1 ngày, cả thế giới có tối đa 160 đống. Nhặt như đồ rơi khác. Cầm phân, trỏ chuột vào cây non, khoai đang lớn hoặc bụi, cây, hoa vừa bị hái: thời gian còn lại giảm một nửa. Bón cho luống đất trống thì luống sẫm lại, khoai trồng ở đó chín nhanh gấp đôi. Xem nhanh: `go run ./cmd/shot -farm -dung`.
- **Tuyết và núi lửa** luôn ở rất xa chỗ xuất phát.
- **Hồi sinh**: chết thì hồi sinh cạnh giường ngủ lần cuối (nếu giường còn), không thì ở chỗ xuất phát.
- **Nhân vật** đầu tròn, thân tròn hơn. **Cá** nhiều màu: cá hề, cá đuôi gai xanh, cá chép vàng, cá koi, cá hồi cầu vồng, cá rô.

| Món mới | Công thức | Tác dụng |
| --- | --- | --- |
| Cửa gỗ | 3 Gỗ | Người đi qua được, thú và thổ dân thì không |
| Sàn gỗ | 1 Gỗ | Lát nền nhà |
| Hàng rào chông | 2 Gỗ, 3 Cành cây | Chặn đường, đâm thú áp sát |
| Bẫy thú | 3 Cành cây, 2 Cỏ khô, 1 Đá | Giữ thú đầu tiên giẫm phải 15 giây (thú to 4 giây); nhấp chuột vào bẫy để đặt lại |
| Đuốc cắm | 1 Đuốc (cầm, nhấp chuột phải vào chỗ muốn cắm) | Cháy 4 phút, sáng và ấm; nhấp chuột vào nó để nhổ lên |
| Bình máu | 2 Hoa anh túc đỏ, 1 Quả mọng, 1 Nấm (gần lửa) | +45 máu |
| Bình máu lớn | 1 Bình máu, 2 Bào tử phát sáng, 1 Hoa lam (gần lửa) | +100 máu, giải độc |
| Băng tơ nhện | 2 Tơ nhện, 1 Cỏ khô | +20 máu, giải độc |
| Giáp da khủng long | 4 Da khủng long, 2 Xương, 1 Tơ nhện | Đỡ 50% sức tấn công, giữ ấm mùa đông |

Xem nhanh: `go run ./cmd/shot -open -tribal`, `-at village`, `-at path`, `-weather storm|rain|wind|tornado|snow`, `-season winter|autumn`, `-zoom -2`.

## Trùm (thế giới mở)

Mỗi thế giới mở có sáu con trùm, mỗi vùng đất hoang một con (`internal/sim/bosses.go`, `bossfight.go`). Trùm sống ở hang ổ sâu trong vùng của nó, xa chỗ xuất phát, và không bao giờ đi xa hang quá 18 ô (Megalodon 34 ô). Nó ngủ cho tới khi có người tới gần, rồi gầm lên và đánh. Ai bỏ đi thì nó về hang và hồi máu. Còn dưới nửa máu thì nó **nổi giận**: gầm lên lần nữa, đánh nhanh hơn và chạy nhanh hơn. Mọi đòn đều có báo trước: trùm há miệng (hoặc chồm lên, cúi đầu, vung đuôi) chừng nửa giây rồi đòn mới tới, nên lùi kịp thì tránh được. Trùm đã hạ thì không bao giờ sống lại (được lưu); bản lưu cũ khi mở sẽ có đủ trùm.

| Trùm | Ở đâu | Đánh thế nào | Máu |
| --- | --- | --- | --- |
| Titanoboa — Chúa rừng rậm | Rừng rậm sâu nhất | Nằm cuộn trong bụi rậm. Cắn (36 máu); thỉnh thoảng quấn lấy người: bị giữ chặt, mỗi giây bị siết 14 máu tới 5 giây. Giữ phím di chuyển để vùng ra nhanh gấp đôi, hoặc đánh nó đủ đau thì nó nhả | 16500 |
| Spinosaurus — Chúa đầm lầy | Ao giữa đầm lầy | Rình dưới nước, chỉ lộ cánh buồm và đôi mắt (không đánh được). Ai tới mép ao trong 5 ô thì lao lên đớp (46 máu); trên cạn cào bằng vuốt (30 máu); lên bờ chừng 14 giây lại lặn xuống. Bơi trong ao của nó cũng bị đớp | 15000 |
| Voi Ma Mút Chúa — Chúa xứ tuyết | Giữa vùng tuyết, có đàn voi đi cùng | Quật ngà (34 máu, hất lùi), lao tới húc từ xa (40 máu, hất văng 3 ô), chồm lên dậm đất trúng mọi người quanh nó 3,6 ô (44 máu). Gầm lên là voi ma mút quanh đó kéo tới đánh cùng | 16500 |
| Carcharodontosaurus — Chúa sa mạc | Giữa sa mạc, tránh xa kim tự tháp | Cắn (38 máu); cúi đầu rồi lao như tên bắn, húc văng người đầu tiên nó đâm phải (44 máu); lao xong phải thở dốc một lúc | 15500 |
| Giganotosaurus — Chúa núi lửa | Vùng tro quanh núi lửa | To hơn bạo chúa. Cắn rất đau (46 máu); quay mình quét đuôi trúng mọi người trong 4,2 ô (36 máu, hất văng) | 16000 |
| Megalodon — Chúa biển sâu | Biển sâu, xa bờ | Lượn vòng quanh thuyền chỉ lộ vây (không đánh được), rồi húc thuyền (thuyền mất 5 độ bền, người 12 máu, người đang bơi 45 máu), nổi lên vài giây đớp người trên thuyền: lúc đó mới đánh được. Thỉnh thoảng nhảy vọt khỏi mặt nước đập xuống cạnh thuyền (thuyền mất 12) | 10000 |

Số máu trên là trước khi giáp đỡ. Với dao hắc diện thạch, giáp răng bạo chúa, 8 Bình máu và 4 Bình máu lớn, một người biết lùi tránh đòn hạ mỗi trùm trong khoảng 1 phút 40 tới 2 phút 40 và vẫn phải uống 2–6 bình; đứng đỡ đòn thì tốn gấp đôi; không giáp, không thuốc thì chết sau vài đòn. Đánh Megalodon cần thuyền buồm (bè gỗ sẽ chìm). Khi trùm đang đánh bạn trong vòng 14 ô, giữa trên màn hình có thanh máu lớn ghi tên nó. Thấy trùm hoặc hang của nó rồi thì bản đồ (M) đánh dấu hang; trùm đã hạ thì có dấu chéo đỏ.

**Chiến lợi phẩm**: xác trùm cho rất nhiều thịt, da, xương (khủng long còn có Da khủng long, Răng bạo chúa; Voi Chúa có lông và ngà), đầu trùm để cắm cọc, và rơi đúng một **món huyền thoại** ngay chỗ nó chết (món này không bao giờ mục):

| Món | Của | Tác dụng |
| --- | --- | --- |
| Giáp vảy Titanoboa | Titanoboa | Giáp tốt nhất: đỡ 80% sức tấn công; không bao giờ trúng độc (nhện, bọ cạp) |
| Giáo răng Spinosaurus | Spinosaurus | Cầm trên tay: giáo mạnh nhất (60), với xa nhất (thêm 1,2 ô) |
| Áo lông Voi Chúa | Voi Ma Mút Chúa | Có trong túi thì không bao giờ bị lạnh: tuyết, đêm đông, gió tuyết |
| Kiếm răng cưa Carcharo | Carcharodontosaurus | Cầm trên tay: 80 mỗi đòn, vung nhanh nhất (0,4 giây) |
| Rìu nanh Giganoto | Giganotosaurus | Cầm trên tay: đòn nặng nhất (150, 0,8 giây); có trong túi thì chặt cây nhanh nhất |
| Vòng răng Megalodon | Megalodon | Đeo (có trong túi): bơi nhanh gấp 1,6 và lâu mệt hơn, chèo và lái thuyền nhanh hơn 25% |

**Phá đảo!** Khi cả sáu trùm đã bị hạ, mọi người chơi đều thấy màn hình "Phá đảo!" một lần: sáu trùm đã hạ, ngày phá đảo, và nút **Tiếp tục chơi**. Đã đóng thì không hiện lại (được lưu theo tên người chơi).

Xem nhanh: `go run ./cmd/shot -open -bosses` (thêm `-bosspose 1`, `2`… để đổi tư thế, `-overlay map -frames 120` để xem dấu hang), `go run ./cmd/shot -victory`.

## Chế tạo

Mở bảng chế tạo (C) để xem mọi công thức và nhấp vào để làm (món xây dựng thì chọn chỗ đặt bằng chuột); túi đồ (Tab) cũng liệt kê công thức của món đang nhấp. Món "cầm trên tay" phải nằm ở ô nhanh đang chọn (1–0).

| Món | Công thức | Tác dụng |
| --- | --- | --- |
| Rìu | 2 Cành cây, 1 Đá | Có trong túi thì chặt được cây, dỡ được tường |
| Cuốc chim | 2 Cành cây, 3 Đá | Có trong túi thì đập đá nhanh gấp đôi |
| Đuốc | 1 Cành cây, 1 Cỏ khô | Cầm trên tay: sáng và ấm quanh mình (cả người đứng gần), cháy 90 giây |
| Giáo | 2 Cành cây, 1 Đá | Cầm trên tay: đánh thú mạnh gấp đôi, với xa hơn nửa ô |
| Cần câu | 3 Cành cây, 2 Cỏ khô | Cầm trên tay mới câu được cá (vẫn tốn 1 Giun) |
| Lửa trại | 2 Gỗ, 2 Cỏ khô, 3 Đá | Ánh sáng và sưởi ấm ban đêm (C) |
| Tường gỗ | 2 Gỗ | Chặn đường người và thú; dỡ bằng rìu, trả lại 1 Gỗ |
| Giường cỏ | 6 Cỏ khô, 2 Gỗ | Nhấp chuột vào giường (hoặc trỏ vào rồi nhấn E) lúc trời tối để ngủ: không bị lạnh, hồi máu nhanh; ai cũng ngủ thì đêm trôi nhanh gấp 8 |
| Thịt / Cá / Quả mọng nướng | 1 món sống | Cầm món sống, đứng gần lửa trại, nhấp chuột phải |

### Thời đồ đá: vật liệu theo bậc

Mỗi bậc nằm thành mạch trong đá ở một vùng riêng, càng tốt càng xa chỗ xuất phát, và chỉ cúp bậc dưới nó mới đào được. Đá lửa, ngọc bích, mã não thô phải mang tới **Lò nung** đang cháy (8 Đá, 2 Gỗ; thêm than hoặc củi) để thành lưỡi hoặc đá mài; hắc diện thạch dùng luôn. Mỗi bậc làm rìu, cúp, cuốc (2 Cành cây + 3 hoặc 2 vật liệu) và dao (1 Cành cây + 2 vật liệu), đánh mạnh hơn bậc dưới.

| Bậc | Đào ở | Cần cúp | Thành |
| --- | --- | --- | --- |
| Đá lửa | Mạch đá lửa trên đồi đá, mỏm nhỏ trong cỏ và rừng; luôn có vài mạch cách chỗ xuất phát 10–34 ô | Cúp đá | Lưỡi đá lửa |
| Ngọc bích | Mạch ngọc bích sâu trong hang, xa lối vào (mang đuốc) | Cúp đá lửa | Ngọc bích mài |
| Mã não | Mạch mã não giữa hoang mạc, hiếm | Cúp ngọc bích | Mã não mài |
| Hắc diện thạch | Vùng tro quanh núi lửa, hiếm | Cúp mã não | (dùng luôn) |

Quặng nằm thành **mạch** 3–8 khối liền nhau, giữa các mạch là khoảng đất trống (`internal/sim/orevein.go`). Than nằm thành vỉa trong hang và trên vùng tro. Thế giới 1024 × 1024 có khoảng 2.000 khối đá lửa, 1.800 ngọc bích, 350 mã não, 450 hắc diện thạch, 3.400 than.

Giáp may từ đồ săn được; chỉ bộ tốt nhất trong túi có tác dụng, đỡ một phần sức tấn công của thú dữ:

| Giáp | Công thức | Đỡ |
| --- | --- | --- |
| Áo da thú | 5 Da thú | 25% |
| Giáp xương | 3 Da thú, 5 Xương | 40% |
| Giáp mã não | 4 Da thú, 2 Xương, 3 Mã não mài | 55% |
| Giáp răng bạo chúa | 4 Da thú, 4 Xương, 2 Răng bạo chúa | 70% |

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
    go run ./cmd/shot -o dev.png -open -devconsole              # bảng lệnh dev đang mở, vài lệnh đã chạy

`SIM_DUMP=1 go test ./internal/sim -run Dump -v` in map ra ASCII để chỉnh thông số sinh map.
