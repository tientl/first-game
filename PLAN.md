# Kế hoạch

Việc cần làm, theo thứ tự ưu tiên hiện tại. Làm xong mục nào thì đánh dấu `[x]` và cập nhật cẩm nang.

## Đồ hoạ Full HD (đang làm)

- [x] Giai đoạn 0: nền tảng vẽ ở độ phân giải màn hình (`internal/client/hd.go`, `HD_ART.md`).
- [ ] Giai đoạn 1: nhân vật và mặt đất (cả hai kiểu: cỏ xanh và đất ấm). Đang dở, đã dừng.
- [ ] Giai đoạn 2: cây, cỏ, hoa, đá, công trình. Đang dở, đã dừng.
- [ ] Giai đoạn 3: khủng long và các sinh vật.
- [ ] Giai đoạn 4: icon, giao diện, hiệu ứng (có thể lấy hiệu ứng từ pixelfx, giấy phép MIT).

## Yêu cầu mới

- [ ] **Tỉ lệ**: dựa vào độ cao nhân vật hiện tại, tăng độ cao của các thứ khác (cây, thú, công trình…) cho cân.
- [ ] **Đuốc**: khi cầm đuốc không hiện ô báo chỗ cắm đuốc nữa, vì đuốc cầm tay đã phát sáng; bấm **chuột phải** để cắm đuốc xuống.
- [ ] **Hoạt ảnh sinh vật**: mỗi sinh vật có đi, chạy, lấy đà, cắn, quất đuôi để tấn công; 8 hoặc 16 khung cho mượt.
- [ ] **Túi đồ**: tăng lên 30 ô; thiết kế lại viền túi đồ cho đẹp hơn.
- [ ] **Rương**: mở rương thì hiện ô chứa đồ trong rương (như túi đồ), không tự rơi đồ ra đất.
- [ ] **Sinh vật mới**: cóc xanh có độc; cóc phát sáng vào ban đêm.
- [ ] **Cung tên**: cung, tên thường, tên độc, tên lửa.
- [ ] **Thời gian**: ngày chậm lại, mỗi ngày 15–20 phút; 1 năm 50 ngày; độ dài ngày và đêm khác nhau theo mùa.
- [ ] **Đĩa bay rơi**:
  - Nằm nghiêng trên mặt đất, có khói và tia điện chập. Đặt ngẫu nhiên trên đảo đầu tiên (đảo người chơi hồi sinh).
  - Cửa bị khoá, cần mật khẩu hoặc chìa khoá. Tìm trong rương ở hang động, chỉ trên đảo đầu tiên, 1 hoặc 2 cái.
  - Vào được thì đổi khung cảnh sang bên trong đĩa bay: máy móc. Có máy quét đồ vật hoặc khoáng sản; quét thì người chơi nhận công thức chế đồ mới dựa vào món đó.
  - Trong đĩa bay có các ô vuông màu đen hiện các con trùm, chỉ thấy viền (bóng) chứ chưa biết là con gì.
  - Vào đó nhặt được một tấm bản đồ: mở ra thì đánh dấu vị trí con trùm đầu tiên trên bản đồ, chỉ hiện "?".
- [ ] **Trứng khủng long, ấp và nuôi khủng long con**:
  - Lấy trộm trứng trong tổ là bị cả đàn đuổi. Mang về ấp cạnh lửa, nở ra khủng long con đi theo bạn và lớn dần thành thú cưỡi.
  - Nuôi từ bé gắn bó hơn thuần thú bằng cho ăn, và cho người chơi lý do quay về căn cứ.
  - **Chỉ thuần hoá được bằng cách ấp trứng** (bỏ cách thuần bằng cho ăn như hiện nay).
- [ ] **Đêm trăng máu, khủng long tấn công trại**:
  - Vài ngày một lần, đêm đỏ rực và bầy raptor hoặc Troodon kéo tới căn cứ.
  - Hàng rào chông, bẫy, tường, đuốc và thú nuôi đã có nhưng chưa có lý do thật để xây; sự kiện này biến chúng thành thứ sống còn.
- [ ] **Cửa hang**: sửa lại cửa vào hang, lối vào nằm ở phía dưới.
- [ ] **Thời tiết theo vùng** (hiện thời tiết chung cho cả thế giới):
  - Sa mạc: không mưa, không tuyết; có bão cát (che tầm nhìn, đi chậm), nắng nóng ban ngày (khát nhanh), đêm lạnh.
  - Vùng tuyết: tuyết và bão tuyết quanh năm, lạnh hơn.
  - Đầm lầy, rừng rậm: mưa nhiều hơn, sương mù.
  - Núi lửa: mưa tro, không tuyết.
  - Biển: giông bão mạnh hơn, sóng lớn ảnh hưởng thuyền.
  - Đồng cỏ: theo mùa như hiện nay.
  - Khi đi từ vùng này sang vùng khác thì thời tiết chuyển dần, không đổi đột ngột.
- [ ] **Khát nước**:
  - Thêm thanh **Khát** cạnh No bụng. Khát giảm dần theo thời gian (nhanh hơn khi chạy, khi nắng nóng ở sa mạc). Khát cạn thì **mất máu** dần.
  - Phải uống **nước sạch**:
    - **Nước mưa**: uống được ngay. Làm vật hứng nước mưa (ví dụ thùng gỗ, lá to) đặt ngoài trời, trời mưa thì đầy dần.
    - **Nước ao, hồ, sông**: múc bằng vật chứa (bình da, bát đất…), phải **nấu sôi** ở lửa trại mới uống được.
    - Uống nước ao chưa nấu: đỡ khát nhưng có thể bị **đau bụng / ngộ độc** (mất máu một lúc).
    - Nước biển: không uống được.
  - Có thể thêm: dừa, dưa, xương rồng cho một ít nước; bình chứa có dung tích (số ngụm).
- [ ] **Chỉnh lại HUD cho hợp lý** (hiện ở 1080p mọi thứ quá nhỏ, bảng phím tắt chiếm nhiều chỗ):
  - **Phóng giao diện theo độ phân giải**: ở 1080p giao diện to gấp đôi (vẽ trên khung 960×540 rồi phóng lên), 720p giữ nguyên; chuột trong túi đồ, rương, chế tạo phải khớp theo.
  - **Bỏ ô hướng dẫn (bảng phím tắt) bên trái màn hình.** Nếu cần xem phím thì bấm **F1** để mở; gợi ý theo ngữ cảnh (ví dụ "Chuột phải: cắm đuốc") hiện 1 dòng nhỏ trên thanh công cụ.
  - **Máu / No bụng / (Khát)**: thanh to hơn, có icon, số dạng "72/100" nằm gọn trong thanh, không tràn chữ.
  - **Trạng thái** (lạnh, trúng độc, dính tơ, đói, khát…): thành các ô icon nhỏ dưới thanh máu, kèm chữ ngắn; câu dài chỉ hiện lúc mới bị.
  - **Đồng hồ ngày / mùa / thời tiết** và **bản đồ nhỏ** góc trên phải gộp thành một khối gọn, có icon thời tiết.
  - **Thanh công cụ**: ô to hơn, số phím rõ hơn, tên món đang cầm hiện rõ.
- [ ] **Thực vật mới** (ưu tiên: tre, bầu, nắp ấm, lanh, thảo dược):
  - Cho khát nước: **tre** (ống đựng nước, vật liệu xây, thân tên); **cây bầu** (quả khô thành bình nước); **cây nắp ấm** (tự hứng nước mưa, uống được ngay; ở đầm lầy, rừng rậm).
  - Cho cung tên: **lanh / gai** (sợi bện dây cung, dây thừng); **dây leo rừng** (dây thừng thô, bẫy); **nấm độc, quả độc** như cà dược (tẩm tên độc; ăn nhầm thì ngộ độc).
  - Chữa bệnh: **lô hội** (chữa bỏng); **thảo dược** (giải độc cóc, nhện, bọ cạp; làm bình máu); **rêu** (đắp vết thương thay băng).
  - Theo vùng: bờ biển có **cây đước**, **rong biển** ăn được; ao và đầm lầy có **hoa súng**, **sen** (lá cho cóc ngồi, hạt sen ăn được); gần nước có **cỏ đuôi ngựa**; đồng cỏ có **mộc lan**; sa mạc có **bụi gai** làm rào tự nhiên; vùng tuyết có **việt quất tuyết** ăn được mùa đông.
  - Trồng trọt thêm: **lúa, ngô hoang, bí** (hiện chỉ có khoai lang).
- [ ] **Sửa phân bố quặng** (hiện mỗi ô của vùng có cùng một tỉ lệ ra quặng, nên quặng rải lẻ tẻ đều khắp vùng, không thành mạch):
  - Quặng mọc thành **mạch / cụm** 3–8 khối liền nhau, giữa các cụm là khoảng trống lớn.
  - Gắn với địa hình: sát vách đá, chân núi, trong hang; càng sâu trong hang càng nhiều và càng hiếm.
  - Độ hiếm tăng theo khoảng cách từ đảo đầu tiên; đảo đầu tiên chỉ có đá và than.
  - Than nằm thành vỉa trong hang và vùng đá.
  - Mỗi loại quặng có dấu hiệu nhận biết từ xa (màu, ánh lấp lánh) để người chơi đi tìm.
  - Đếm thực tế (thế giới 1024×1024): than 23.112, đá lửa 241, ngọc bích 21.587, mã não 413, hắc diện thạch 419. Đá lửa (bậc thấp nhất) lại hiếm nhất; ngọc bích và than quá nhiều vì hang mê cung rất rộng.
  - Mục tiêu: đá lửa vài nghìn (có cả gần đảo đầu tiên); ngọc bích 1.000–2.000, ở sâu trong hang; than ít hơn, thành vỉa; mã não và hắc diện thạch giữ khoảng 400.
  - **Thêm kim cương**: bậc 5, quý nhất, hiếm hơn hắc diện thạch (khoảng 50–100 mạch). Chỉ ở chỗ sâu nhất của hang dưới đảo núi lửa hoặc gần hang ổ trùm; cần cúp hắc diện thạch mới đào được; lấp lánh trong bóng tối. Làm công cụ, kiếm, giáp bậc 5 mạnh nhất (dưới đồ huyền thoại của trùm).
  - **Quặng hồi lại** (hiện đào hết là mất vĩnh viễn, kể cả tảng đá lớn):
    - Đá, than, đá lửa: **hồi ở chỗ cũ** sau vài ngày trong game (ví dụ 5–10 ngày), chỉ khi không có người đứng gần.
    - Ngọc bích, mã não, hắc diện thạch, kim cương: game giữ tổng số mạch mỗi loại; mạch bị đào thì **mọc mạch mới ở chỗ khác** trong đúng vùng của nó, xa người chơi (như thú vật hồi lại). Càng quý càng hồi chậm.
- [ ] **Bản đồ lớn hơn: 2048 × 2048 ô** (hiện 1024 × 1024: khoảng 252 nghìn ô đất, 56 đảo / 28 đảo lớn; đi bộ băng ngang ~4 phút, thuyền buồm ~2,5 phút):
  - Gấp 4 diện tích: đi bộ băng ngang ~8,5 phút, thuyền buồm ~5 phút; nhiều đảo hơn, biển rộng hơn.
  - Chỉ áp dụng cho thế giới tạo mới; thế giới đã lưu giữ cỡ cũ.
  - Kiểm tra: thời gian tạo thế giới, bộ nhớ server, cỡ bản lưu (~4 lần), bản đồ (M) và bản đồ nhỏ vẫn mượt; số thú, làng, hang, trùm, quặng tăng theo diện tích.
