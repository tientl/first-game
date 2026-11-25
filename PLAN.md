# Kế hoạch

Việc cần làm, chia theo nhóm. Trong mỗi nhóm, mục trên làm trước. Làm xong mục nào thì đánh dấu `[x]` và cập nhật cẩm nang.

## Mốc DEMO (làm trước hết)

Bản demo cho người khác chơi thử. Chỉ làm những mục dưới đây; mọi thứ khác trong kế hoạch để sau demo.

- [x] **Hướng nhân vật theo chuột chỉ khi ngắm**: bình thường nhân vật quay theo hướng đi (WASD), không quay theo con trỏ nữa. Chỉ khi dùng kỹ năng có hướng (cung, ném, sau này phép…) thì **giữ chuột** để ngắm: lúc đó nhân vật mới quay theo con trỏ, thả chuột thì bắn/ném.
- [ ] **Đồ hoạ Full HD — ưu tiên số 1** (bắt buộc trong demo; mục 2). Chất lượng hình ảnh là thứ người chơi thấy đầu tiên, nên làm trước và làm kỹ nhất: mọi thứ người chơi thấy trong demo đều vẽ lại ở Full HD, không để nửa cũ nửa mới:
  - Nhân vật (đủ kiểu ngoại hình, giáp, vũ khí, cưỡi, bơi, ngủ) và mặt đất (cỏ xanh và đất ấm).
  - Cây, cỏ, hoa, đá, quặng, công trình.
  - Mọi sinh vật có trong thế giới demo, kèm hoạt ảnh nhiều khung; Titanoboa và Megalodon to, đẹp nhất.
  - Icon túi đồ, giao diện, hiệu ứng (lửa, chém, va chạm, nước…).
  - **Chuẩn chất lượng**: mỗi nhóm (nhân vật, mặt đất, cây, từng nhóm sinh vật, icon) làm xong thì gửi ảnh so sánh để duyệt; chưa đạt thì làm lại trước khi sang nhóm khác.
  - Nếu thiếu lượt dùng: **bớt số loài thú trong demo chứ không hạ chất lượng** (ưu tiên các loài ở đảo đầu tiên, rừng rậm, biển).
- [ ] **Cảm giác đánh nhau** (mục 7): khựng lúc trúng đòn, thú bật lùi, chớp trắng, rung màn hình khi đòn mạnh (hiệu ứng chém, va chạm lấy từ pixelfx); **giơ khiên đỡ**; thú **báo trước đòn** (lấy đà rõ ràng trước khi cắn, quất đuôi, lao).
- [x] **Chế độ dev** (mục 1) — để test nhanh trước khi phát hành.
- [ ] **Giao diện** (mục 3): HUD gọn, bỏ ô hướng dẫn (F1), túi 30 ô, rương có ô đồ, chuột phải cắm đuốc.
- [x] **Sửa phân bố quặng** (mục 4): đá lửa đủ nhiều, ngọc bích và than bớt; chưa cần kim cương.
- [ ] **Thời gian** (mục 4): ✅ ngày 16 phút, mùa 12 ngày, ngày và đêm theo mùa; còn **cửa hang** (miệng hang quay xuống phía dưới màn hình, xem mục 4).
- [x] **Chỉ 2 trùm** trong demo: các trùm khác tắt (không sinh ra, không hiện trên bản đồ); màn hình "Phá đảo!" tính theo 2 trùm. Hai trùm có thanh máu và nhạc chiến đấu (mục 8; nhạc ✅ `bossmusic.go`). Trùm được chọn: **Titanoboa** (rừng rậm) và **Megalodon** (biển sâu).
- [ ] **Hướng dẫn chơi** vài phút đầu.
- [ ] **Menu cài đặt**: âm lượng nhạc / tiếng, và phần **Hiển thị**:
  - Chế độ màn hình: Toàn màn hình / Toàn màn hình không viền / Cửa sổ (hiện chỉ có F11).
  - Độ phân giải: danh sách theo màn hình (1280×720, 1600×900, 1920×1080, 2560×1440…). Sắc nét nhất ở 1080p và các mức gấp nguyên lần (4K).
  - Chất lượng vẽ: Cao (100%) / Trung bình (75%) / Thấp (50%) cho máy yếu.
  - Cỡ giao diện: Tự động / 100% / 150% / 200%.
  - Giới hạn khung hình: 30 / 60 / 120 / không giới hạn, kèm VSync.
  - Thu phóng camera (đã có phím = / −) đưa vào menu.
  - Mặc định tự động: độ phân giải màn hình, toàn màn hình, chất lượng Cao, cỡ giao diện Tự động. Cài đặt được lưu lại.
- [ ] **Chạy thử dài (soak test)** — làm ngay sau Full HD: để game chạy 1–2 giờ tự động (nhân vật đi lang thang, nhiều ngày trong game), ghi bộ nhớ và thời gian khung hình; bộ nhớ không được tăng dần, khung hình không chậm dần.
- [ ] **Chạy lại một ván cố định sau mỗi tính năng** — làm ngay sau Full HD: cùng seed, cùng chuỗi thao tác, chụp ảnh ở các mốc bằng `cmd/shot`, so với lần trước để phát hiện thay đổi ngoài ý muốn.
- [ ] **Kiểm tra cân bằng tiến trình**: test tự động cho thời gian lên từng bậc công cụ, tài nguyên đủ cho từng bậc (như phát hiện đá lửa hiếm hơn hắc diện thạch); cùng bảng đấu tay đôi đã có.
- [ ] **Phiếu góp ý cho người chơi thử**: chỗ bị kẹt, chỗ chán, chỗ khó hiểu, chơi bao lâu thì bỏ, thích nhất gì.
- [ ] **Checklist phát hành**: build sạch Windows và Linux, test qua hết; chế độ dev tắt hẳn, không còn giá trị test hay log gỡ lỗi; lưu/tải được ở mọi tình huống (bơi, cưỡi, đánh trùm, trong hang); hướng dẫn chơi đã thử với người mới; chữ đã soát; màn hình ghi công kèm giấy phép (Ebitengine, pixelfx…); lỗi văng thì ghi file log; đánh số phiên bản và gắn tag Git.
- [ ] **Chơi thử, sửa lỗi**, build `.exe` cho Windows và bản Linux.

## 1. Công cụ dev (làm đầu tiên)

- [x] **Chế độ dev** (cờ `-dev` cho `cmd/game`), để test nhanh mọi tính năng:
  - Bỏ qua menu, vào thẳng thế giới mới (nhỏ, seed cố định, bản lưu riêng).
  - Bất tử, không đói, không khát, không lạnh (bật/tắt được).
  - Bảng lệnh (phím `~`): `give <đồ> <số>`, `tp <làng|trùm N|hang|kim tự tháp|x y>`, `spawn <thú>`, `time <giờ|đêm|ngày>`, `season <mùa>`, `weather <thời tiết>`, `speed <hệ số>`, `kill` (thú quanh), `heal`.
  - Hiện hết bản đồ (bỏ vùng chưa khám phá).
  - Không bao giờ bật trong bản phát hành: chỉ khi chạy với `-dev`.

## 2. Đồ hoạ

### Full HD (đang làm)

- [x] Giai đoạn 0: nền tảng vẽ ở độ phân giải màn hình (`internal/client/hd.go`, `HD_ART.md`).
- [x] Giai đoạn 1: nhân vật (8 khung đi, giáp, đấm, bơi, ngủ, màn hình tạo nhân vật) và mặt đất (cả hai kiểu: cỏ xanh và đất ấm; `SetGroundStyle`, lưu trong cài đặt).
- [x] Giai đoạn 2: cây, cỏ, hoa, đá, công trình (vẽ lại ở HD). Còn tinh chỉnh tự động, cần vẽ tay sau: quặng, lò nung, lều, cây ăn thịt, chông, kim tự tháp, cửa hang. Lửa, thuyền, icon vẫn hình cũ.
- [ ] Giai đoạn 3: khủng long và các sinh vật. **Xong phần khủng long** (16 loài vẽ từ PNG, 8 khung đi/chạy, 8 khung vỗ cánh, lấy đà trước đòn — hình HD tạo tự động từ hình cũ, chưa vẽ tay). Còn: thú vẽ bằng code (thỏ, rắn, cua, cá sấu, voi ma mút, hổ răng kiếm, gấu, nhện, thổ dân, khỉ, bọ cạp, cá, giun), quái biển, 6 trùm; và sim phải trì hoãn đòn đánh để lấy đà là báo trước thật (hiện trúng đòn ngay khi bắt đầu lấy đà).
- [ ] Giai đoạn 4: icon, giao diện, hiệu ứng (có thể lấy hiệu ứng từ pixelfx, giấy phép MIT).

### Tỉ lệ và hoạt ảnh

- [ ] **Tỉ lệ**: dựa vào độ cao nhân vật hiện tại, tăng độ cao của các thứ khác (cây, thú, công trình…) cho cân.
- [ ] **Hoạt ảnh sinh vật**: mỗi sinh vật có đi, chạy, lấy đà, cắn, quất đuôi để tấn công; 8 hoặc 16 khung cho mượt.

## 3. Giao diện và điều khiển

- [ ] **Chỉnh lại HUD cho hợp lý** (hiện ở 1080p mọi thứ quá nhỏ, bảng phím tắt chiếm nhiều chỗ):
  - **Phóng giao diện theo độ phân giải**: ở 1080p giao diện to gấp đôi (vẽ trên khung 960×540 rồi phóng lên), 720p giữ nguyên; chuột trong túi đồ, rương, chế tạo phải khớp theo.
  - **Bỏ ô hướng dẫn (bảng phím tắt) bên trái màn hình.** Nếu cần xem phím thì bấm **F1** để mở; gợi ý theo ngữ cảnh (ví dụ "Chuột phải: cắm đuốc") hiện 1 dòng nhỏ trên thanh công cụ.
  - **Máu / No bụng / (Khát)**: thanh to hơn, có icon, số dạng "72/100" nằm gọn trong thanh, không tràn chữ.
  - **Trạng thái** (lạnh, trúng độc, dính tơ, đói, khát…): thành các ô icon nhỏ dưới thanh máu, kèm chữ ngắn; câu dài chỉ hiện lúc mới bị.
  - **Đồng hồ ngày / mùa / thời tiết** và **bản đồ nhỏ** góc trên phải gộp thành một khối gọn, có icon thời tiết.
  - **Thanh công cụ**: ô to hơn, số phím rõ hơn, tên món đang cầm hiện rõ.
- [ ] **Túi đồ**: tăng lên 30 ô; thiết kế lại viền túi đồ cho đẹp hơn.
- [ ] **Rương**: mở rương thì hiện ô chứa đồ trong rương (như túi đồ), không tự rơi đồ ra đất.
- [ ] **Đuốc**: khi cầm đuốc không hiện ô báo chỗ cắm đuốc nữa, vì đuốc cầm tay đã phát sáng; bấm **chuột phải** để cắm đuốc xuống.

## 4. Thế giới và môi trường

- [ ] **Bản đồ lớn hơn: 2048 × 2048 ô** (hiện 1024 × 1024: khoảng 252 nghìn ô đất, 56 đảo / 28 đảo lớn; đi bộ băng ngang ~4 phút, thuyền buồm ~2,5 phút):
  - Gấp 4 diện tích: đi bộ băng ngang ~8,5 phút, thuyền buồm ~5 phút; nhiều đảo hơn, biển rộng hơn.
  - Chỉ áp dụng cho thế giới tạo mới; thế giới đã lưu giữ cỡ cũ.
  - Kiểm tra: thời gian tạo thế giới, bộ nhớ server, cỡ bản lưu (~4 lần), bản đồ (M) và bản đồ nhỏ vẫn mượt; số thú, làng, hang, trùm, quặng tăng theo diện tích.
- [ ] **Thời gian**: ngày chậm lại, mỗi ngày 15–20 phút; 1 năm 50 ngày; độ dài ngày và đêm khác nhau theo mùa.
- [ ] **Thời tiết theo vùng** (hiện thời tiết chung cho cả thế giới):
  - Sa mạc: không mưa, không tuyết; có bão cát (che tầm nhìn, đi chậm), nắng nóng ban ngày (khát nhanh), đêm lạnh.
  - Vùng tuyết: tuyết và bão tuyết quanh năm, lạnh hơn.
  - Đầm lầy, rừng rậm: mưa nhiều hơn, sương mù.
  - Núi lửa: mưa tro, không tuyết.
  - Biển: giông bão mạnh hơn, sóng lớn ảnh hưởng thuyền.
  - Đồng cỏ: theo mùa như hiện nay.
  - Khi đi từ vùng này sang vùng khác thì thời tiết chuyển dần, không đổi đột ngột.
- [ ] **Cửa hang**: miệng hang **quay xuống phía dưới màn hình** (về phía người nhìn): một ụ đá / vách đá có mặt đứng hướng xuống dưới, ở chân vách là vòm cửa tối có lối đất dẫn ra; chỉ đi vào được từ phía dưới (hai bên và phía sau là đá chặn). Hiện tại trông như một cái hố nhìn từ trên xuống.
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
- [ ] **Thực vật mới** (ưu tiên: tre, bầu, nắp ấm, lanh, thảo dược):
  - Cho khát nước: **tre** (ống đựng nước, vật liệu xây, thân tên); **cây bầu** (quả khô thành bình nước); **cây nắp ấm** (tự hứng nước mưa, uống được ngay; ở đầm lầy, rừng rậm).
  - Cho cung tên: **lanh / gai** (sợi bện dây cung, dây thừng); **dây leo rừng** (dây thừng thô, bẫy); **nấm độc, quả độc** như cà dược (tẩm tên độc; ăn nhầm thì ngộ độc).
  - Chữa bệnh: **lô hội** (chữa bỏng); **thảo dược** (giải độc cóc, nhện, bọ cạp; làm bình máu); **rêu** (đắp vết thương thay băng).
  - Theo vùng: bờ biển có **cây đước**, **rong biển** ăn được; ao và đầm lầy có **hoa súng**, **sen** (lá cho cóc ngồi, hạt sen ăn được); gần nước có **cỏ đuôi ngựa**; đồng cỏ có **mộc lan**; sa mạc có **bụi gai** làm rào tự nhiên; vùng tuyết có **việt quất tuyết** ăn được mùa đông.
  - Trồng trọt thêm: **lúa, ngô hoang, bí** (hiện chỉ có khoai lang).

## 5. Sinh tồn

- [ ] **Khát nước**:
  - Thêm thanh **Khát** cạnh No bụng. Khát giảm dần theo thời gian (nhanh hơn khi chạy, khi nắng nóng ở sa mạc). Khát cạn thì **mất máu** dần.
  - Phải uống **nước sạch**:
    - **Nước mưa**: uống được ngay. Làm vật hứng nước mưa (ví dụ thùng gỗ, lá to) đặt ngoài trời, trời mưa thì đầy dần.
    - **Nước ao, hồ, sông**: múc bằng vật chứa (bình da, bát đất…), phải **nấu sôi** ở lửa trại mới uống được.
    - Uống nước ao chưa nấu: đỡ khát nhưng có thể bị **đau bụng / ngộ độc** (mất máu một lúc).
    - Nước biển: không uống được.
  - Có thể thêm: dừa, dưa, xương rồng cho một ít nước; bình chứa có dung tích (số ngụm).
- [ ] **Dụng cụ nấu ăn và món mới** (hiện chỉ nướng từng miếng cạnh lửa trại). Làm trước: nồi đất, giàn phơi, 3–4 món canh/súp có hiệu ứng:
  - Dụng cụ:
    - **Nồi đất** (đất sét, nung ở lò): đun sôi nước ao cho sạch (gắn với khát nước); nấu canh, súp, hầm từ nhiều nguyên liệu.
    - **Xiên nướng / giàn nướng** (gỗ, cành): đặt trên lửa trại, nướng 3–4 miếng cùng lúc.
    - **Chảo đá**: rán trứng (gắn với trứng khủng long), áp chảo.
    - **Cối giã** (đá): giã thảo dược thành thuốc, giã hạt thành bột.
    - **Lò đất nướng bánh** (đá, đất sét): nướng bánh từ bột (gắn với lúa, ngô).
    - **Giàn phơi / hun khói** (gỗ, dây thừng): thịt khô, cá khô để được lâu.
    - **Bát, bình nước** (đất nung, quả bầu, da): đựng canh, đựng nước uống.
  - Món nấu có hiệu ứng tạm thời: canh thịt rau (hồi máu nhanh), súp nóng (chống lạnh), cá nướng thảo mộc (chạy nhanh), thịt hầm khủng long (no lâu, thêm sức đánh), trà thảo dược (giải độc), nước dừa / nước ép quả (đỡ khát, thêm chút máu).
  - **Muối**: phơi nước biển trên giàn ở bờ biển; ướp thịt để được lâu, nấu món ngon hơn.
  - **Đồ ăn ôi thiu** (để sau, chỉ ở độ khó Thành thạo và Chúa tể): thịt, cá sống để lâu thì hỏng; thịt khô, muối, hun khói giữ được lâu.
- [ ] **Quần áo và trang bị** (hiện giáp chỉ cần nằm trong túi, game tự chọn bộ tốt nhất; áo lông trong túi thì chống lạnh; không có ô mặc đồ). Làm trước: ô trang bị + hiện trên người, chỉ số Ấm / Mát / Giáp, túi đeo lưng, khiên tròn:
  - **6 ô trang bị**: Mũ, Áo, Quần, Giày, Áo choàng, Đeo thêm (vòng cổ, túi đeo). Mặc gì thì hiện đúng trên nhân vật Viking (mũ sắt có thanh che mũi, áo choàng lông, ủng…).
  - **Độ bền**: mòn khi bị đánh, sửa bằng nguyên liệu ở bàn chế tạo.
  - **Chỉ số**: Giáp (đỡ đòn), Ấm (tuyết, đêm đông), Mát (sa mạc, khát chậm hơn), Chống mưa, Tốc độ (đồ nặng đi chậm).
  - **Đồ theo vùng**: áo choàng / mũ / ủng lông (tuyết); khăn trùm, áo vải lanh, dép (sa mạc); áo tơi lá (mưa); giày da cá sấu (đi nhanh ở đầm lầy); áo ngụy trang lá (thú để ý muộn).
  - **Túi đeo lưng**: thêm ô túi đồ. **Khiên tròn Viking**: giơ lên đỡ đòn. **Mũ sắt Viking**: đỡ đòn vào đầu.
  - Để sau: **nhuộm màu áo** (màu từ hoa, quả); **bộ đồ đủ món** có thêm hiệu ứng (ví dụ đủ bộ da bạo chúa thì thú nhỏ sợ).

## 6. Sinh vật và thuần thú

- [ ] **Trứng khủng long, ấp và nuôi khủng long con**:
  - Lấy trộm trứng trong tổ là bị cả đàn đuổi. Mang về ấp cạnh lửa, nở ra khủng long con đi theo bạn và lớn dần thành thú cưỡi.
  - Nuôi từ bé gắn bó hơn thuần thú bằng cho ăn, và cho người chơi lý do quay về căn cứ.
  - **Chỉ thuần hoá được bằng cách ấp trứng** (bỏ cách thuần bằng cho ăn như hiện nay).
- [ ] **Sinh vật mới**: cóc xanh có độc; cóc phát sáng vào ban đêm.

## 7. Chiến đấu và sự kiện

- [ ] **Cảm giác đánh nhau** (trong demo): khựng lúc trúng đòn (hit-stop), thú bật lùi, chớp trắng, rung màn hình khi đòn mạnh hoặc chí mạng; hiệu ứng từ pixelfx: `melee/claw-swipe` (vết cào) cho raptor, gấu, hổ răng kiếm, khỉ, vượn chúa; `melee/sword-slash`, `thrust`, `spear-sweep`, `heavy-cleave` cho kiếm, giáo, rìu; `impacts/hit-spark`, `crit-hit`, `hit-flash` khi trúng đòn và chí mạng; `water/splash` cho Megalodon; giơ khiên đỡ đòn; thú báo trước đòn để người chơi đọc và né; chặn mẹo đánh một đòn rồi lùi.
- [ ] **Cung tên**: cung, tên thường, tên độc, tên lửa.
- [ ] **Đêm trăng máu, khủng long tấn công trại**:
  - Vài ngày một lần, đêm đỏ rực và bầy raptor hoặc Troodon kéo tới căn cứ.
  - Hàng rào chông, bẫy, tường, đuốc và thú nuôi đã có nhưng chưa có lý do thật để xây; sự kiện này biến chúng thành thứ sống còn.

## 8. Trùm

- [ ] **Yêu cầu chung cho mọi trùm** (6 trùm hiện có và 2 trùm mới):
  - **To**: trùm phải to hơn hẳn thú thường; ít nhất cỡ bạo chúa, trùm mới (vượn, bọ cạp chúa) to gấp 2–3 lần. Rà lại cỡ 6 trùm hiện có cho cân.
  - **Thanh máu**: đã có thanh máu trùm ở trên cùng màn hình; thêm tên trùm, chia vạch theo giai đoạn (ví dụ dưới 50% máu thì đổi cách đánh, nổi giận), hiệu ứng khi máu tụt.
  - **Nhạc chiến đấu** (hiện chưa có; nhạc chỉ có ngày / đêm / hang / menu): nhạc riêng khi đánh trùm, tổng hợp bằng code như nhạc hiện tại: trống dồn, kèn trầm, dây căng; mỗi trùm một chủ đề (Titanoboa: trống rừng; Spinosaurus: nước, trầm; Voi Chúa: tù và; Carcharo: sa mạc; Giganoto: núi lửa, nặng; Megalodon: biển sâu; vượn chúa: trống bộ lạc; bọ cạp chúa: Ai Cập cổ). Nhạc dâng lên khi trùm thức dậy, dồn dập hơn khi trùm dưới 50% máu, tắt dần khi bỏ chạy, có đoạn nhạc chiến thắng khi hạ trùm.
- [ ] **Trùm thứ 7: vượn khổng lồ ở Đảo Khỉ** (không dùng tên "King Kong" vì là nhân vật có bản quyền; dựa trên **Gigantopithecus**, loài vượn khổng lồ có thật thời tiền sử):
  - **Đảo Khỉ**: một đảo rừng rậm mới, nhiều cây chuối, đàn khỉ đông; giữa đảo là hang ổ trên núi đá có tượng đá, xương thú.
  - Tên gợi ý: **Gigantopithecus — Chúa Đảo Khỉ**. Cao gấp 3 bạo chúa, lông đen xám, ngực bạc.
  - Đánh: đấm hai tay xuống đất (sóng chấn động, hất văng mọi người quanh nó), vồ bắt người ném đi, nhổ cây / nhặt tảng đá ném từ xa, đấm ngực gầm lên gọi đàn khỉ tới ném quả và cào. Máu thấp thì leo lên vách đá hồi sức, rồi nhảy xuống đập đất.
  - Chiến lợi phẩm huyền thoại gợi ý: **Găng tay vượn chúa** (đấm tay không mạnh như vũ khí, mang vác nặng không chậm) hoặc **Bờm vượn chúa** (thú nhỏ không dám lại gần).
  - Thêm vào màn hình "Phá đảo!" (7 trùm) và vào các ô bóng trùm trong đĩa bay.
- [ ] **Trùm thứ 8: bọ cạp khổng lồ trong kim tự tháp** (hiện kim tự tháp chỉ là khối đá có rương ở cửa, bọ cạp thường canh bên ngoài):
  - **Vào được bên trong kim tự tháp**: đi vào cửa thì đổi cảnh sang lòng kim tự tháp (như hang, đĩa bay): hành lang tối, bẫy (gai bật, đá lăn, cát lún), cần đuốc; sâu nhất là phòng mộ có quan tài và kho báu.
  - Tên gợi ý: **Bọ cạp Chúa — Chúa kim tự tháp**: to như bạo chúa, vỏ đen ánh vàng, đuôi phát sáng xanh độc.
  - Đánh: kẹp hai càng (giữ người lại, siết máu), chích đuôi (độc nặng, mất máu lâu), phun độc thành vũng trên sàn, chui xuống cát rồi trồi lên dưới chân người chơi, gọi bọ cạp con. Vỏ cứng: phải đánh vào đuôi hoặc lúc nó ngóc lên mới trúng mạnh.
  - Chiến lợi phẩm huyền thoại gợi ý: **Giáp vỏ bọ cạp** (miễn độc, phản lại một phần sát thương) hoặc **Dao đuôi bọ cạp** (mỗi đòn làm thú trúng độc).
  - Đưa vào màn hình "Phá đảo!" và các ô bóng trùm trong đĩa bay.

## 10. Đồ trang trí (sau demo)

- [ ] **Đồ trang trí** (hiện chỉ có đầu thú cắm cọc, sàn gỗ, đuốc cắm, hoa trồng từ hạt). Chỉ để trang trí, **không có hệ thống "Ấm cúng" / "Nghỉ ngơi"**. Làm trước: thảm lông, giá vũ khí, đầu thú treo tường, đèn nấm, bàn ghế:
  - Nội thất: bàn, ghế, ghế dài, kệ gỗ (bày đồ lên); giá vũ khí, giá giáp (treo kiếm, giáo, khiên, giáp, hiện đúng hình); thảm lông thú (gấu, hổ răng kiếm, da khủng long, mỗi loại một hoa văn); giường lông.
  - Đèn: đèn dầu, lồng đèn treo, chậu than; đèn nấm phát sáng; đèn đom đóm trong lọ.
  - Trưng bày: đầu thú treo tường; bệ trưng bày đầu trùm, món huyền thoại; bể cá thả cá câu được. Để sau (cùng phần hang): bộ xương khủng long, hoá thạch.
  - Viking và bộ lạc: cờ, khiên tròn sơn màu treo tường, tượng gỗ chạm, cột vật tổ tự làm, chuông gió xương, dây treo răng thú.
  - Ngoài trời: chậu cây, bồn hoa, hàng rào gỗ trang trí, cổng gỗ, đường lát đá, bù nhìn rơm.

## 9. Khám phá

- [ ] **Nhiệm vụ từ làng thổ dân** (sau demo): "mang 5 da raptor", "dẹp tổ nhện trong hang", "hộ tống thợ săn"…; thưởng đồ, công thức, thiện cảm với làng.
- [ ] **Sổ hành trình** (sau demo): các cột mốc ("đóng thuyền đầu tiên", "vào được đĩa bay", "hạ Titanoboa"…), cho người mới biết làm gì tiếp.
- [ ] **Điểm thú vị trên mỗi đảo** (sau demo): xác tàu đắm, tàn tích đá cổ, trại thợ săn bỏ hoang, hang bí mật sau thác nước, **bản đồ kho báu** (dấu X ở đảo khác), đồ hiếm chỉ có ở đó.
- [ ] **Trao đổi với thổ dân** (sau demo): đổi da, thịt, quặng lấy đồ hiếm, hạt giống lạ, bản đồ; làng có **thiện cảm** (thân thì giúp, thù thì tấn công).
- [ ] **Nhịp độ**: 10 phút đầu có chuỗi mục tiêu nhỏ (nhặt cành → rìu → lửa → qua đêm đầu tiên); xen kẽ căng và nghỉ (ngày khám phá, đêm nguy hiểm, sự kiện bất ngờ như trăng máu, bão, thú lạ).
- [ ] **Đĩa bay rơi**:
  - Nằm nghiêng trên mặt đất, có khói và tia điện chập. Đặt ngẫu nhiên trên đảo đầu tiên (đảo người chơi hồi sinh).
  - Cửa bị khoá, cần mật khẩu hoặc chìa khoá. Tìm trong rương ở hang động, chỉ trên đảo đầu tiên, 1 hoặc 2 cái.
  - Vào được thì đổi khung cảnh sang bên trong đĩa bay: máy móc. Có máy quét đồ vật hoặc khoáng sản; quét thì người chơi nhận công thức chế đồ mới dựa vào món đó.
  - Trong đĩa bay có các ô vuông màu đen hiện các con trùm, chỉ thấy viền (bóng) chứ chưa biết là con gì.
  - Vào đó nhặt được một tấm bản đồ: mở ra thì đánh dấu vị trí con trùm đầu tiên trên bản đồ, chỉ hiện "?".

## 11. Chơi chung qua mạng (sau cùng)

- [ ] **Co-op qua mạng**: kết nối TCP giữa client và server (nền server/client đã có); menu "Tạo phòng" / "Vào phòng" (nhập IP); hướng dẫn dùng Tailscale / ZeroTier để chơi qua Internet không cần mở cổng; bản lưu thuộc chủ phòng, người vào lại bằng đúng tên giữ nguyên nhân vật; Windows và Linux chơi chung được. Sau này có thể dùng mạng Steam.
