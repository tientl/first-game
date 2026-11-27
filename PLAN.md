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
- [x] **Giao diện** (mục 3): HUD gọn, bỏ ô hướng dẫn (F1), túi 30 ô, rương có ô đồ, chuột phải cắm đuốc.
- [x] **Sửa phân bố quặng** (mục 4): đá lửa đủ nhiều, ngọc bích và than bớt; chưa cần kim cương.
- [x] **Thời gian** (mục 4): ✅ ngày 16 phút, mùa 12 ngày, ngày và đêm theo mùa; ✅ **cửa hang** (miệng hang quay xuống phía dưới màn hình, xem mục 4).
- [x] **Chỉ 2 trùm** trong demo: các trùm khác tắt (không sinh ra, không hiện trên bản đồ); màn hình "Phá đảo!" tính theo 2 trùm. Hai trùm có thanh máu và nhạc chiến đấu (mục 8; nhạc ✅ `bossmusic.go`). Trùm được chọn: **Titanoboa** (rừng rậm) và **Megalodon** (biển sâu).
- [x] **Gặp trùm phải ấn tượng** (trước đây trùm đứng trơ trọi ngoài trời, gặp thì nhạt):
  - **Hang ổ thật**: Titanoboa nằm cuộn trong một hố rễ cây khổng lồ / phế tích đá giữa rừng rậm, quanh có xương thú, da rắn lột, cây đổ; Megalodon ở một vùng biển sâu tối màu có xác tàu, xương cá voi, đàn cá tản ra.
  - **Dấu hiệu khi lại gần**: xương, vết trườn, cây gãy, chim bay tán loạn, thú nhỏ bỏ chạy; với Megalodon là vây thấp thoáng, mặt nước gợn, cá nhảy.
  - **Trùm ẩn mình** cho tới khi tới gần: Titanoboa nằm im lẫn trong lá, chỉ thấy mắt; Megalodon chỉ là bóng tối dưới nước.
  - **Màn ra mắt**: camera khựng và kéo về phía trùm, màn hình tối viền, chữ to "TITANOBOA — Chúa rừng rậm", trùm gầm / phóng lên, nhạc chiến đấu bắt đầu.
  - **Đấu trường**: khi trận bắt đầu, lối ra bị chắn (rễ, dây leo; với Megalodon là vùng nước xoáy) cho tới khi hạ trùm hoặc người chơi chạy thật xa.
  - Lúc không đánh, trùm có hoạt động: Titanoboa săn thú quanh hang, cuộn ngủ; Megalodon lượn vòng, đớp cá.
- [ ] **Hướng dẫn chơi** vài phút đầu.
- [ ] **Menu cài đặt**: âm lượng nhạc / tiếng, và phần **Hiển thị**:
  - Chế độ màn hình: Toàn màn hình / Toàn màn hình không viền / Cửa sổ (hiện chỉ có F11).
  - Độ phân giải: danh sách theo màn hình (1280×720, 1600×900, 1920×1080, 2560×1440…). Sắc nét nhất ở 1080p và các mức gấp nguyên lần (4K).
  - Chất lượng vẽ: Cao (100%) / Trung bình (75%) / Thấp (50%) cho máy yếu.
  - Cỡ giao diện: Tự động / 100% / 150% / 200%.
  - Giới hạn khung hình: 30 / 60 / 120 / không giới hạn, kèm VSync.
  - Thu phóng camera (đã có phím = / −) đưa vào menu.
  - Mặc định tự động: độ phân giải màn hình, toàn màn hình, chất lượng Cao, cỡ giao diện Tự động. Cài đặt được lưu lại.
- [x] **Bộ chiêu tấn công cho từng loài** (mục 7): mỗi loài 2–4 chiêu, chiêu nào cũng có lấy đà rõ (đòn nhẹ 0,4–0,5 giây, đòn mạnh 0,8–1 giây, đòn đặc biệt 1,2–1,5 giây kèm vùng đỏ dưới đất), sát thương chỉ tính lúc ra đòn, đánh xong khựng 0,5–1 giây cho người chơi phản công. ✅ `sim/skills.go` (bảng chiêu của 17 loài, vùng đánh hình quạt/tròn/đường, hồi chiêu, bầy thay phiên, đòn mạnh làm khựng thú nhỏ), `sim/skillai.go`, `sim/status.go` (choáng, chảy máu, chậm, bị đè: bấm đánh liên tục để vùng ra), `client/dinoskills.go`, `client/critskills.go` (dáng lấy đà, ra đòn, hồi), `client/telegraphzone.go` (vùng đỏ tạm), `client/skillsound.go`; dev: `fight <thú>`, `skill <thú> [chiêu]`; `cmd/shot -skills` quay GIF từng loài. Giao thức 49.
- [ ] **Trận trùm Titanoboa và Megalodon**: chiêu báo trước, đổi đòn theo giai đoạn máu.
- [ ] **Ghi công và giấy phép** (bắt buộc trước Steam): màn hình "Ghi công" trong menu (tiếng CC BY trong `CREDITS.md`, Ebitengine, phông chữ, pixelfx nếu dùng) và file `LICENSES.txt` kèm game (nguyên văn giấy phép Apache, MIT, BSD, OFL của các thư viện và phông).
- [ ] **Kiểm tra trước khi cho chơi thử**: đo RAM và khung hình thực tế trên máy Windows (bộ nhớ đệm mặt đất, cây, tiếng, trùm, chuyển động); thời gian tải thế giới 2048; test tải bản lưu cũ qua từng đời (save 1 → mới nhất) không hỏng.

## 1. Công cụ dev (làm đầu tiên)

- [x] **Chế độ dev** (cờ `-dev` cho `cmd/game`), để test nhanh mọi tính năng:
  - Bỏ qua menu, vào thẳng thế giới mới (nhỏ, seed cố định, bản lưu riêng).
  - Bất tử, không đói, không khát, không lạnh (bật/tắt được).
  - Bảng lệnh (phím `~`): `give <đồ> <số>`, `tp <làng|trùm N|hang|kim tự tháp|x y>`, `spawn <thú>`, `time <giờ|đêm|ngày>`, `season <mùa>`, `weather <thời tiết>`, `speed <hệ số>`, `kill` (thú quanh), `heal`.
  - Hiện hết bản đồ (bỏ vùng chưa khám phá).
  - Không bao giờ bật trong bản phát hành: chỉ khi chạy với `-dev`.
- [x] **Xây tự do** trong chế độ dev (`sim/devbuild.go`, `client/buildpalette.go`; mặc định bật, lệnh `freebuild on|off`, nhãn "DEV · XÂY TỰ DO", lưu trong `dev-save.json`): chế tạo mọi món không cần nguyên liệu, không cần lửa / lò, không mất gì (thẻ chế tạo sáng hết, có huy hiệu DEV); **B** mở bảng xây với mọi thứ đặt được (tường, cửa, sàn, rào, giường, rương, lều, lửa, đuốc, lò, đồ nấu trên lửa, luống đất, hạt giống, hoa, đầu thú, đồ trang trí), chọn rồi đặt liên tục, giữ kéo để đặt một dãy, luật mặt đất vẫn giữ; **Shift + chuột phải** gỡ ngay, không trả lại gì; chặt, đập, đào một nhát. Server không ở chế độ dev thì từ chối, bản lưu thường không bao giờ có. Giao thức 47.

## 2. Đồ hoạ

### Full HD (đang làm)

- [x] Giai đoạn 0: nền tảng vẽ ở độ phân giải màn hình (`internal/client/hd.go`, `HD_ART.md`).
- [x] Giai đoạn 1: nhân vật (8 khung đi, giáp, đấm, bơi, ngủ, màn hình tạo nhân vật) và mặt đất (cả hai kiểu: cỏ xanh và đất ấm; `SetGroundStyle`, lưu trong cài đặt).
- [x] Giai đoạn 2: cây, cỏ, hoa, đá, công trình (vẽ lại ở HD). Còn tinh chỉnh tự động, cần vẽ tay sau: quặng, lò nung, lều, cây ăn thịt, chông, kim tự tháp. Lửa, thuyền, icon vẫn hình cũ.
- [x] Giai đoạn 3: khủng long và các sinh vật. 16 loài khủng long và Ankylosaurus vẽ tay lại ở HD (`internal/dinopaint`, mắt mở, đùi chuyển động theo chân), voi ma mút, hổ răng kiếm, quái biển và trùm vẽ HD. Còn thú nhỏ và thổ dân: xem dòng dưới.
- [ ] Giai đoạn 4: icon, giao diện, hiệu ứng (có thể lấy hiệu ứng từ pixelfx, giấy phép MIT).
  - [ ] **Biểu tượng**: mới 20 / ~150 món vẽ HD (`iconhd.go`); còn lại chia nhóm vẽ tiếp.
  - [ ] **Giao diện HD** (đang làm): thanh máu, ô đồ, túi, rương, chế tạo, nấu ăn, bản đồ nhỏ, thanh trùm, menu vẽ ở độ phân giải thật; phông chữ vector có đủ dấu tiếng Việt (giấy phép OFL); khung kiểu da thuộc, gỗ, đá, xương; chữ nổi trong thế giới nhỏ gọn, chỉ hiện khi trỏ chuột.
  - [ ] **Đồ cầm tay HD** (đang làm): rìu, cuốc, giáo, dao, kiếm, đuốc, cần câu, bát… vẽ HD, xoay mượt khi vung.
  - [ ] **Lửa, đuốc, hiệu ứng** từ pixelfx (làm chung với Cảm giác đánh nhau); thuyền vẽ lại.
  - [ ] **Bộ lọc ánh sáng bật mặc định ở mức nhẹ** (`postfx.go`), gộp với kết quả thử tông màu COMFY52 (đang thử 3 bản: hiện tại / chỉnh màu theo COMFY52 / ép đúng 52 màu).
- [ ] **Thú nhỏ, thổ dân và làng HD**: thỏ, cua, cá, khỉ, nhện, giun, rắn, bọ cạp, người và nhà thổ dân vẽ lại cho ngang khủng long.
- [ ] **Cây cỏ và mặt đất** (đang làm): mỗi loại 4–6 kiểu, mọc thành đám, lá rụng và rêu dưới tán, cỏ lay theo gió và rạp khi đi qua, màu theo mùa, vẽ lại thân cây tuế, bớt giun.
- [ ] **Biển và vách đá**: san hô thành cụm gọn, sóng biển sâu vẽ HD, vách đá cao có khối như tranh mẫu.
- [ ] **Quặng đá lửa** ở đồi đá bớt dày (giữ đá lửa vẫn là quặng nhiều nhất).
- [ ] **Chuyển động nhân vật**: làm lại dáng chạy (chúi người, tay vung, chân đá ngược, nhún) và đi bộ (đang làm); hết lộ chân khi bơi ở nước sâu (đang làm); cưỡi thú nhìn từ trước và sau (phải vẽ thêm góc nhìn cho thú cưỡi, chờ quyết).
- [ ] **Xoay hướng đồ xây dựng**: phím R hoặc lăn chuột khi đặt, hình mờ hiện đúng hướng, hướng lưu trong game; cửa, giường, lều, hàng rào, tường, bàn ghế, đồ nấu ăn, bàn chế tạo, rương; cả bảng xây tự do của dev.
- [ ] **Hang mới** (đang làm): mỗi đảo một hệ hang, khoang lớn nối đường hầm rộng 3–6 ô, hai tầng, khoang theo chủ đề (pha lê, hồ ngầm, nấm, ổ nhện, hang gấu, hoá thạch, tranh vách) và theo đảo phía trên (rễ cây, sa thạch và lăng mộ, ống dung nham, hang băng, hang ngập nước).

### Tỉ lệ và hoạt ảnh

- [x] **Nhân vật có khớp tay chân** (giữ cách vẽ bằng code, như khủng long `dinolegs.go`): vai–khuỷu–cổ tay, hông–gối–cổ chân xoay quanh khớp; đi, chạy 8–12 khung, bàn chân không trượt, tay vung ngược chân; chém, đâm, bổ, đấm có lấy đà – ra đòn – thu về; áo choàng, tóc, râu đung đưa trễ nhịp; đủ 4 hướng, mọi kiểu ngoại hình, giáp, đồ cầm tay khớp với bàn tay.
- [ ] **Tỉ lệ**: dựa vào độ cao nhân vật hiện tại, tăng độ cao của các thứ khác (cây, thú, công trình…) cho cân.
  - [x] **Cây, cây cỏ, công trình, đá**: lấy người chơi (cao 93 px ở 1080p) làm thước, vẽ lại ở cỡ mới (thêm lá, vỏ cây, ván, đá chứ không phóng to): cây thường 3–5 lần người, cây lá kim 3–5, bách tán, gỗ đỏ, bụt mọc 5–6, cây cổ thụ khổng lồ 7 lần (rễ bạnh, vỏ nứt, rêu, hốc cây); bụi ngang hông, cỏ ngang đùi, hoa và cây trồng ngang gối; cửa ra vào cao 1,2 lần người (cửa gỗ, lều, cửa hang, kim tự tháp); tường rào, đuốc, cột cờ, cột tô-tem, cây đước, cây phát sáng theo cỡ người. Vẫn đứng trên ô cũ, va chạm không đổi. Cây, mái lều che người hay thú đứng sau thì mờ đi. Cây to vẽ khi cần (`lazyart.go`). Bảng so sánh: `go run ./cmd/shot -w 1920 -h 1080 -scalesheet`.
  - [x] **Rừng thưa theo cỡ cây mới** (`woods.go`, `thinwoods.go`): cây to gấp 2–3 lần nên rừng cũ thành bức tường lá (tán phủ 85–99% màn hình 1080p, không thấy đất, không thấy người). Thế giới mới (StyleWoods) đặt cây theo khoảng cách tán (Poisson-disc, tán to trước rồi cây non xen giữa): cây lá rộng, gỗ đỏ, bụt mọc to cách nhau 3,2–4 ô, cây lá kim, bách 2 ô, dương xỉ thân gỗ, dừa dưới 2 ô; bìa rừng thưa, vào trong dày dần, có trảng trống, đường mòn xuyên mọi khu rừng lớn, dưới tán có cỏ cao, bụi, cành khô, hoa, nấm trên khúc gỗ mục. Rừng có hai kiểu theo một nhiễu chậm, chuyển dần sang nhau: **rừng dày** (~35% diện tích rừng, tán phủ ~60–70%, cây sát nhau) và **rừng bình thường** (tán ~35–45%); bìa ~30%; đảo rừng rậm, đảo khỉ nghiêng về dày, đảo nhà và quanh chỗ xuất phát nghiêng về bình thường (nhiều cây non hơn để có gỗ). Bản đồ rừng: `go run ./cmd/worldstat -woodsmap rung.png`. Cây ít đi khoảng 5 lần nên mỗi cây cho nhiều gỗ hơn (3, cây to 10). Tổ ong theo số cây: khoảng 1 tổ / 45 cây rừng. Save cũ (bản 19 trở về trước) được tỉa một lần khi mở theo cùng bản đồ dày / thường (dịu hơn), không đụng cây người chơi trồng, cây đang chặt, tổ ong đã lấy mật, cây gần công trình, hang, kim tự tháp, hang ổ trùm; tổ ong thừa bị bỏ bớt. Bị tán che thì nhân vật hiện viền sáng (`hiddenself.go`). Đo tán: `go run ./cmd/shot -open -canopy -real`; xem save cũ sau khi tỉa: `-oldsave` hoặc `-load <bản sao save>`.
  - [x] **Rừng mọc lại** (`regrow.go`, `saplingart.go`): cây bị chặt còn gốc 1 ngày → cây non (vẽ mới) 1,5 ngày → cây nhỏ → sau 2 ngày nữa thành cây to nếu trước là cây to (một ngày 16 phút): cây nhỏ về lại sau 2,5 ngày, cây to sau 4,5 ngày. Rừng nhớ chỗ từng cây lúc tạo thế giới; một lượt quét chậm (cả bản đồ mỗi 2 ngày, vài tile mỗi tick) cho cây non mọc lại vào chỗ trống nếu đủ chỗ cho cây to và không gần công trình, làng, hang, kim tự tháp, hang ổ trùm. Cây lớn lên đôi khi có tổ ong mới. Xem: `go run ./cmd/shot -open -fell 8 -days 3 -real`, dải các giai đoạn `-regrow`.
  - [x] **Bãi biển rộng ra** (`beaches.go`, kiểu thế giới mới `StyleBeaches`; thế giới cũ tạo lại y như trước): bờ cát không còn là dải 1–3 ô mà thành bãi biển sâu 5–9 ô (đo ở 1024: trung vị 7 ô, ~70% bờ cát sâu từ 5 ô; trước: 2 ô), rộng dần hẹp dần dọc bờ; bờ hướng ra biển khơi và vịnh của đảo nhà rộng tới 10–12 ô, vũng nhỏ hẹp hơn. Cát lấn vào đất liền chứ không lấn ra biển (đất/biển, số đảo giữ nguyên), đất sau bãi được giữ bằng phẳng nên bãi không dựng vào vách đá; bờ vách đá vẫn không có cát (cát thu hẹp dần rồi hết khi tới vách), bờ đước, đầm lầy, tuyết, tro vẫn hẹp. Bãi rộng có cồn cát và cỏ cồn ở phía sau, dừa mọc ở phía sau bãi chứ không sát nước, gỗ trôi dạt (cành cây vẽ bạc màu trên cát) và nghêu dọc ngấn nước; client vẽ dải cát ướt ở mép nước, ngấn rong biển dạt, vỏ sò, không còn bờ đất dựng dưới bãi cát. Nước nông vẽ lại theo một độ sâu liền mạch (`terrainsea.go`): từ xanh nhạt trên cát sát bãi qua ngọc lam tới xanh dương ở mép thềm rồi xanh thẫm ngoài khơi, mép mềm, hết các mảng vuông xanh lốm đốm ở bãi cạn (do alpha bị tràn số), ánh sáng gợn đáy nhẹ thành lưới, sóng vỗ bờ vẽ mịn ở độ phân giải HD. Save bản 21. Xem: `go run ./cmd/shot -open -size 1024 -at beach`, `go run ./cmd/worldstat` (dòng beaches).
  - [ ] **Thú**: chưa đổi cỡ. Hiện so với người: thằn lằn săn mồi 1,1 lần, hổ răng kiếm 0,9, gấu 0,8, ba sừng 1,5, phiến lưng 1,7, voi ma mút 2,4, bạo chúa 2,3, cổ dài 3,5 (cổ dài và phiến lưng còn nhỏ so với thật).
- [ ] **Hoạt ảnh sinh vật**: mỗi sinh vật có đi, chạy, lấy đà, cắn, quất đuôi để tấn công; 8 hoặc 16 khung cho mượt.

## 3. Giao diện và điều khiển

- [x] **Chỉnh lại HUD cho hợp lý** (hiện ở 1080p mọi thứ quá nhỏ, bảng phím tắt chiếm nhiều chỗ):
  - **Phóng giao diện theo độ phân giải**: ở 1080p giao diện to gấp đôi (vẽ trên khung 960×540 rồi phóng lên), 720p giữ nguyên; chuột trong túi đồ, rương, chế tạo phải khớp theo.
  - **Bỏ ô hướng dẫn (bảng phím tắt) bên trái màn hình.** Nếu cần xem phím thì bấm **F1** để mở; gợi ý theo ngữ cảnh (ví dụ "Chuột phải: cắm đuốc") hiện 1 dòng nhỏ trên thanh công cụ.
  - **Máu / No bụng / (Khát)**: thanh to hơn, có icon, số dạng "72/100" nằm gọn trong thanh, không tràn chữ.
  - **Trạng thái** (lạnh, trúng độc, dính tơ, đói, khát…): thành các ô icon nhỏ dưới thanh máu, kèm chữ ngắn; câu dài chỉ hiện lúc mới bị.
  - **Đồng hồ ngày / mùa / thời tiết** và **bản đồ nhỏ** góc trên phải gộp thành một khối gọn, có icon thời tiết.
  - **Thanh công cụ**: ô to hơn, số phím rõ hơn, tên món đang cầm hiện rõ.
- [x] **Túi đồ**: tăng lên 30 ô; thiết kế lại viền túi đồ cho đẹp hơn.
- [x] **Rương**: mở rương thì hiện ô chứa đồ trong rương (như túi đồ), không tự rơi đồ ra đất.
- [x] **Đuốc**: khi cầm đuốc không hiện ô báo chỗ cắm đuốc nữa, vì đuốc cầm tay đã phát sáng; bấm **chuột phải** để cắm đuốc xuống.

## 4. Thế giới và môi trường

- [x] **Bản đồ lớn hơn: 2048 × 2048 ô** (hiện 1024 × 1024: khoảng 252 nghìn ô đất, 56 đảo / 28 đảo lớn; đi bộ băng ngang ~4 phút, thuyền buồm ~2,5 phút):
  - Gấp 4 diện tích: đi bộ băng ngang ~8,5 phút, thuyền buồm ~5 phút; nhiều đảo hơn, biển rộng hơn.
  - Chỉ áp dụng cho thế giới tạo mới; thế giới đã lưu giữ cỡ cũ.
  - Kiểm tra: thời gian tạo thế giới, bộ nhớ server, cỡ bản lưu (~4 lần), bản đồ (M) và bản đồ nhỏ vẫn mượt; số thú, làng, hang, trùm, quặng tăng theo diện tích.
  - ✅ Cỡ **Khổng lồ (2048)** chọn được ở màn hình tạo thế giới (phím C), mặc định vẫn **Lớn (1024)**. Đo (`go run ./cmd/worldstat -size N`): 1024 tạo ~1,4 giây, server ~58 MB, bản lưu ~20 MB; 2048 tạo ~6 giây, ~227 MB, bản lưu ~76 MB (gấp ~3,9 lần), tải lại ~7 giây. Thú ở xa mọi người chơi "ngủ gật" (bước 1/8 số tick) nên một tick ở 2048 chỉ ~11 ms. Bản đồ vẽ song song, nét gấp 4; sương mù vẽ lần đầu ngoài vòng lặp game. Trùm vẫn mỗi loài một con.
- [x] **Đảo và địa hình phong phú hơn** (kiểu thế giới mới `StyleIsles`, `sim/isles.go`, `coast.go`, `heights.go`, `rivers.go`; thế giới cũ tạo lại y như trước):
  - Quần đảo thiết kế: đảo nhà ở giữa (đồng cỏ, một ngọn đồi, vịnh kín), quanh đó các đảo lớn mỗi đảo một tính cách (rừng rậm, châu thổ đầm lầy, đảo khỉ có núi đá, dãy núi tuyết có vịnh hẹp, sa mạc cao nguyên, núi lửa), đảo vừa, đảo nhỏ, một đảo san hô vòng có phá bên trong; bờ có vịnh, mũi đất, bán đảo, eo biển giữa các đảo.
  - Ô mới: **biển nông** (bơi và đi thuyền được; rạn san hô, phá, bãi cạn), **sông** (bơi qua), **chỗ lội** (đi bộ qua), **thác**, **vách đá**. Mỗi ô có thêm một byte địa hình: tầng cao 0–3, biến thể (đồng hoa, trảng rừng, rừng dày / thưa, mỏm đá, cồn cát, đá vụn, bãi bùn, bãi đá tảng, dốc lên, …), mép mềm, đất ẩm.
  - Vách đá giữa các tầng, có dốc lên đủ để đi khắp mỗi đảo; sông từ chỗ cao chảy ra biển, có thác, hồ, châu thổ. Hình vẽ hiện là tạm (`client/terrainfeatures.go`), chờ vẽ đẹp.
- [x] **Thời gian**: ngày chậm lại, mỗi ngày 15–20 phút; 1 năm 50 ngày; độ dài ngày và đêm khác nhau theo mùa. (Đã làm: ngày 16 phút, mùa 12 ngày, ngày và đêm theo mùa.)
- [ ] **Camera**: lia theo người mượt, phóng to thu nhỏ mượt, rung màn hình khi có sự kiện lớn (gầm của bạo chúa, trùm, đòn mạnh).
- [ ] **Mưa và đêm đẹp hơn**: hạt mưa, vũng nước, ánh chớp; ban đêm có ánh trăng, bóng tối mềm quanh nguồn sáng.
- [ ] **Thời tiết theo vùng** (hiện thời tiết chung cho cả thế giới):
  - Sa mạc: không mưa, không tuyết; có bão cát (che tầm nhìn, đi chậm), nắng nóng ban ngày (khát nhanh), đêm lạnh.
  - Vùng tuyết: tuyết và bão tuyết quanh năm, lạnh hơn.
  - Đầm lầy, rừng rậm: mưa nhiều hơn, sương mù.
  - Núi lửa: mưa tro, không tuyết.
  - Biển: giông bão mạnh hơn, sóng lớn ảnh hưởng thuyền.
  - Đồng cỏ: theo mùa như hiện nay.
  - Khi đi từ vùng này sang vùng khác thì thời tiết chuyển dần, không đổi đột ngột.
- [x] **Cửa hang**: miệng hang **quay xuống phía dưới màn hình** (về phía người nhìn): một ụ đá / vách đá có mặt đứng hướng xuống dưới, ở chân vách là vòm cửa tối có lối đất dẫn ra; chỉ đi vào được từ phía dưới (hai bên và phía sau là đá chặn). ✅ Vẽ lại ở HD (`caveart.go`): ụ đá phủ rêu, vòm cửa có đá lanh tô, dây leo, lối đất; lối ra trong hang là vòm sáng ở phía trên màn hình trên một dãy bậc. Đá chặn hai bên và phía sau đã có test (`TestCaveMouthOpensSouth`).
- [x] **Hang động tự nhiên** (kiểu thế giới mới `StyleCaves`, sau `StyleWilds`; thế giới cũ tạo lại y như trước; bản lưu 25, giao thức 48). Trước đây là một mê cung khắp bản đồ: lối đi 2 ô, góc vuông, ngõ cụt vô tận, mọi cửa hang dẫn vào cùng một mê cung — vào là lạc.
  - [x] **Bước 1 — cấu trúc** (`sim/cavesys.go`, `cavegen.go`, `cavedig.go`, `cavedress.go`, `caveprops.go`; client `cavebake.go`, `cavethings.go`, `cavemap.go`, `cavelights.go`): **mỗi đảo một hệ thống hang** nằm ngay dưới đảo, to theo diện tích đảo (đảo rừng 100 nghìn ô: 27 gian trên, 16 gian dưới; đảo nhỏ: một gian sảnh); mọi cửa hang của đảo dẫn vào hệ thống đó; vài đường hầm bí mật dài dưới đáy biển nối tầng dưới của hai đảo gần nhau. **Gian rộng** 15–40 ô, tường tròn lởm chởm, cột đá và cụm măng đá (chặn đường, đào ra đá); **đường hầm uốn lượn 3–6 ô** (không chỗ nào hẹp dưới 3 ô), có vòng, nhánh phụ ngắn và cuối nhánh có rương; **một lối chính** từ cửa hang tới gian đích (rương báu) hiện thành vệt đất mòn sáng hơn. **Hai tầng**: tầng trên ngay dưới đảo, sảnh cửa hang có ánh nắng rọi xuống, có than; tầng dưới tối hơn, xuống bằng **hố có dây thừng** ở cuối tầng trên, leo dây để lên, có ngọc bích (và mã não dưới sa mạc, hắc diện thạch dưới núi lửa) và gấu. **Nhện chỉ ở tổ nhện** (khoảng 120 con ở 1024, trước 6.400), **gấu ở hang gấu**. Vách hang có mặt đá cao ngang người rưỡi, chia mặt phẳng lấy sáng; nền đá có sỏi, chỗ ẩm, vũng nước; đá theo đảo phía trên (đá vôi, đá rêu, sa thạch, bazan, băng, bùn). **Bản đồ (M)** và bản đồ nhỏ vẽ riêng từng tầng hang theo phần đã đi qua, đánh dấu lối lên và dây thừng đã thấy. **Đuốc cắm trong hang cháy mãi** để đánh dấu đường. Test: `go test ./internal/sim -run TestCaves`; xem: `go run ./cmd/shot -open -size 1024 -at cavein|cavedown|caveup`, bản đồ `-overlay map -reveal -mapzoom 3`.
  - [x] **Bước 2 — chủ đề** (`sim/cavethemes.go`; client `cavezones.go`, `caveprops.go`): **mỗi hang mang chất của đảo phía trên** (theo vùng trên đảo: dưới sườn rừng của đảo núi lửa là hang rêu, dưới miệng núi là ống dung nham): đảo rừng — rễ cây buông qua mép vách, rêu, **suối ngầm** có chỗ lội; sa mạc — sa thạch nhiều lớp, **hầm mộ cổ** lát đá có quan tài, chum và rương, bọ cạp; núi lửa — **ống dung nham** (dung nham chặn đường, đứng sát bị bỏng, phát sáng), đá bazan, cột bazan, mảnh hắc diện thạch; tuyết — **hang băng** vách băng trong, nhũ băng, **hồ đóng băng** đi được; đầm lầy — **hang ngập** vũng nước có cá sấu; đồng cỏ — đá vôi hiền lành, **hồ ngầm**. Gian đích ở cuối lối chính là gian riêng của đảo. **Gian chủ đề** khắp nơi: hang pha lê ngọc bích phát sáng (có mạch ngọc ở tầng dưới), hồ ngầm có cá mù (câu được), rừng nấm phát sáng (nấm ăn được), tổ nhện giăng tơ (đi vào bị tơ giữ chân), hang gấu đầy xương, chỗ dơi đậu (dơi bay), **bộ xương khủng long hoá thạch** (đào lấy xương một lần), **tranh vẽ trên vách** (xem: người đi săn, đàn thú, ngôi sao lửa rơi xuống đảo giữa biển, hoặc rương báu kèm hướng đi tới gian đích). Cá mù, cá sấu, bọ cạp, nhện, gấu sống trong đúng gian của mình. Test: `go test ./internal/sim -run 'TestCaves'`; xem từng gian: `go run ./cmd/shot -open -size 1024 -at cavezone:crystal|lake|mushroom|nest|den|roost|fossil|paintings|tomb|lava|frozen|flooded|stream`, `-at caverock:limestone`.
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
- [x] **Thực vật mới** (ưu tiên: tre, bầu, nắp ấm, lanh, thảo dược). ✅ 18 loài vẽ HD (`floraart.go`, `florawet.go`, `floracrop.go`; sim `flora.go`): tre, bầu, nắp ấm, lanh, dây leo, nấm độc, cà độc, lô hội, thảo dược, rêu, đước, rong biển, hoa súng, sen, cỏ đuôi ngựa, lúa / ngô / bí (mọc hoang và trồng được). Đã dùng được: ăn, hồi máu, giải độc, bện dây thừng, băng rêu, tường / sàn / bè tre, bình máu thảo dược, nướng ngô / bí / cốm. Còn chờ hệ thống khác: khát nước (ống tre, bình bầu, nước nắp ấm), cung tên (dây cung, tên độc). **Chưa làm**: mộc lan, bụi gai sa mạc, việt quất tuyết.
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
- [x] **Dụng cụ nấu ăn và món mới** (hiện chỉ nướng từng miếng cạnh lửa trại). Làm trước: nồi đất, giàn phơi, 3–4 món canh/súp có hiệu ứng:
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
- [x] **Lều lá**: khung chữ A bằng cành, lợp lá to (dương xỉ trên cỏ, lá rộng trong rừng và đầm lầy, tàu dừa trên cát và sa mạc, cành thông phủ tuyết trên tuyết); làm bằng tay từ 6 cành cây, 4 cỏ khô, 2 sợi lanh, mang trong túi, cầm trên tay rồi dựng (không dựng trên đá hay chỗ lội). Dùng như giường: ngủ qua đêm, làm nơi hồi sinh, rừng không mọc lại sát bên; người ngủ nằm trong lều, lá và khung che lên đầu và chân. Ban ngày cầm rìu thì dỡ được lều (và cả giường), lấy lại một nửa vật liệu.
- [ ] **Quần áo và trang bị** (hiện giáp chỉ cần nằm trong túi, game tự chọn bộ tốt nhất; áo lông trong túi thì chống lạnh; không có ô mặc đồ). Làm trước: ô trang bị + hiện trên người, chỉ số Ấm / Mát / Giáp, túi đeo lưng, khiên tròn:
  - **6 ô trang bị**: Mũ, Áo, Quần, Giày, Áo choàng, Đeo thêm (vòng cổ, túi đeo). Mặc gì thì hiện đúng trên nhân vật Viking (mũ sắt có thanh che mũi, áo choàng lông, ủng…).
  - **Độ bền**: mòn khi bị đánh, sửa bằng nguyên liệu ở bàn chế tạo.
  - **Chỉ số**: Giáp (đỡ đòn), Ấm (tuyết, đêm đông), Mát (sa mạc, khát chậm hơn), Chống mưa, Tốc độ (đồ nặng đi chậm).
  - **Đồ theo vùng**: áo choàng / mũ / ủng lông (tuyết); khăn trùm, áo vải lanh, dép (sa mạc); áo tơi lá (mưa); giày da cá sấu (đi nhanh ở đầm lầy); áo ngụy trang lá (thú để ý muộn).
  - **Túi đeo lưng**: thêm ô túi đồ. **Khiên tròn Viking**: giơ lên đỡ đòn. **Mũ sắt Viking**: đỡ đòn vào đầu.
  - Để sau: **nhuộm màu áo** (màu từ hoa, quả); **bộ đồ đủ món** có thêm hiệu ứng (ví dụ đủ bộ da bạo chúa thì thú nhỏ sợ).

## 6. Sinh vật và thuần thú

- [x] **Thú theo đảo, đảo to hơn, đảo nhiều vùng** (kiểu thế giới mới `StyleWilds`, save bản 23; thế giới cũ tạo lại y như trước): ✅ `sim/wilds.go`, `sim/wildzones.go`, `sim/wildlairs.go`, `sim/isles.go`.
  - **Thú theo đảo và vòng xa gần** (bảng `wildRules`): đảo nhà chỉ có thú hiền (thỏ, Thescelosaurus, Ornithomimus, vài đàn ba sừng và gai lưng ở phía xa; vài nhóm ngay quanh chỗ xuất phát), ban ngày không có thú săn mồi, không hang rắn, không cá sấu, không thằn lằn cổ dài ven bờ (Troodon ban đêm vẫn như cũ). Đảo gần: Acheroraptor, Anzu, thằn lằn bay, cá sấu đầm lầy; Dakotaraptor chỉ sâu trong rừng. Đảo xa (sa mạc, núi lửa, tuyết): đàn cổ dài, bạo chúa (mỗi đảo sa mạc / núi lửa lớn ít nhất một con), Dakotaraptor, Quetzalcoatlus. Mỗi loại đảo có loài riêng; trong đảo thú theo vùng (thú ăn cỏ ở rìa cỏ sa mạc, raptor trong rừng sườn núi lửa, hổ răng kiếm trong rừng thông). Khủng long bọc giáp là thú ăn cỏ của đảo rừng / đồng cỏ, không ở đảo nhà. Thú ăn cỏ đông gấp ~5 thú săn mồi; cổ dài ~80 (gấp 4); đảo từ 1.500 ô đều có đàn thú; thú hồi lại đúng chỗ đã đặt nên đảo nhà dọn sạch vẫn chỉ có thú hiền. Thương long và mực khổng lồ ít đi ~75%, chỉ ở biển sâu xa đảo nhà; thằn lằn cổ dài gấp đôi, cá nhiều hơn. Hang động không đổi (việc của phần hang).
  - **Đảo ít hơn, to hơn**: 6 đảo lớn xếp vòng quanh đảo nhà, to gấp ~1,2–2 lần (ở 1024: 40–100 nghìn ô, 6–11 màn hình ngang), đảo nhà gấp ~1,2–1,5; hai đảo sát nhau luôn có eo biển; ít đảo nhỏ; biển vẫn ~55% thế giới; đảo to có nhiều đồi, sông hơn. Đảo rừng của Titanoboa là đảo to nhất, hang ổ nằm sâu giữa lõi rừng dày; mỗi trùm trên cạn ở đảo to nhất của loại mình; Megalodon ở một góc biển sâu để trống, cách đất liền 30+ ô.
  - **Mỗi đảo nhiều vùng**: loại đất chính chiếm ~50–65%, còn lại 2–4 vùng phụ đúng chỗ: đồng cỏ có rừng, đồi đá, bãi lầy ven sông; rừng rậm có trảng cỏ, sườn núi đá, bờ lầy; đầm lầy có gò rừng, đồng cỏ ướt; sa mạc có ốc đảo (ao, cỏ), rìa cỏ khô có cây, núi mặt bàn; núi lửa có rừng rậm ở sườn thấp; núi tuyết có rừng thông, lãnh nguyên ven biển.
  - Xem: `go run ./cmd/worldstat -isles -fauna -zones` (cỡ đảo, tỉ lệ vùng, đảo của trùm, thú theo vòng và loại đảo), `-faunamap`, `-islesmap`.
- [ ] **Trứng khủng long, ấp và nuôi khủng long con**:
  - Lấy trộm trứng trong tổ là bị cả đàn đuổi. Mang về ấp cạnh lửa, nở ra khủng long con đi theo bạn và lớn dần thành thú cưỡi.
  - Nuôi từ bé gắn bó hơn thuần thú bằng cho ăn, và cho người chơi lý do quay về căn cứ.
  - **Chỉ thuần hoá được bằng cách ấp trứng** (bỏ cách thuần bằng cho ăn như hiện nay).
- [x] **Sinh vật mới**: cóc xanh có độc; cóc phát sáng vào ban đêm. ✅ `sim/toad.go`, `client/toadart.go`: sống ở đầm lầy và ven ao, nhảy 8 khung, ngồi trên lá súng / lá sen; chạm vào cóc xanh (hoặc đấm tay không) là trúng độc, rơi da cóc và tuyến độc; cóc phát sáng toả sáng ban đêm, rơi tuyến phát sáng.
- [x] **Khủng long bọc giáp (Ankylosaurus)**. ✅ `sim/ankylo.go`, `client/ankylo.go`, `client/ankylorig.go`, `dinopaint/ankylo.go`, `dinopaint/armour.go`: thú ăn cỏ thấp, rộng, lưng phủ tấm xương có gờ, gai nón dọc sống lưng, gai lưỡi dao dọc sườn và vai, đuôi cứng kết thúc bằng chùy xương hai núm; từng nhóm 1–3 con ở rừng và đồng cỏ, đôi khi bên sông hay trong đầm lầy (thế giới cũ có dần khi thú được bổ sung). Không tự tấn công: ai lại quá gần hoặc đánh nó thì nó quay đuôi về phía người đó, khuỵu xuống lấy đà rồi quật chùy ngang (báo trước rõ), đòn nặng nhất trong các loài ăn cỏ, hất văng người chơi, làm cả bạo chúa choáng. Giáp đỡ nửa đòn vào lưng, sườn, đuôi; đánh từ phía đầu (đầu, bụng mềm) thì trúng trọn; vung xong nó quay người chậm. Rơi thịt, da, xương và **tấm giáp xương** (dùng làm Giáp xương). Thuần bằng quả mọng (cầm quả mọng thì nó cho lại gần), cưỡi được (chậm). Giao thức 45, bản lưu 22.
- [ ] **Thú non**: mọi loài có con non (voi ma mút con, hổ răng kiếm con, khủng long con, thỏ con, gấu con…), nhỏ hơn, đầu to, chân ngắn, chạy lon ton theo mẹ trong đàn. Đánh con non thì thú mẹ nổi giận lao vào. Con non lớn dần thành con trưởng thành sau vài ngày. Khủng long con nở từ trứng (mục trên) dùng chung hình con non.

## 7. Chiến đấu và sự kiện

- [ ] **Cảm giác đánh nhau** (trong demo): khựng lúc trúng đòn (hit-stop), thú bật lùi, chớp trắng, rung màn hình khi đòn mạnh hoặc chí mạng; hiệu ứng từ pixelfx: `melee/claw-swipe` (vết cào) cho raptor, gấu, hổ răng kiếm, khỉ, vượn chúa; `melee/sword-slash`, `thrust`, `spear-sweep`, `heavy-cleave` cho kiếm, giáo, rìu; `impacts/hit-spark`, `crit-hit`, `hit-flash` khi trúng đòn và chí mạng; `water/splash` cho Megalodon; giơ khiên đỡ đòn; thú báo trước đòn để người chơi đọc và né; chặn mẹo đánh một đòn rồi lùi.
- [ ] **Cung tên**: cung, tên thường, tên độc, tên lửa.
- [ ] **Đêm trăng máu, khủng long tấn công trại**:
  - Vài ngày một lần, đêm đỏ rực và bầy raptor hoặc Troodon kéo tới căn cứ.
  - Hàng rào chông, bẫy, tường, đuốc và thú nuôi đã có nhưng chưa có lý do thật để xây; sự kiện này biến chúng thành thứ sống còn.

## 8. Trùm

- [ ] **Hang ổ cho 4 trùm còn lại** (hiện chỉ Titanoboa và Megalodon có hang ổ, ẩn mình, dấu hiệu báo trước, màn mở đầu, đấu trường; 4 trùm kia đứng giữa trời nên tạm bật chế độ demo mặc định, `cmd/game -demo=false` để có đủ 6):
  - **Spinosaurus**: khúc sông lớn hoặc châu thổ đầm lầy; nằm chìm dưới nước, chỉ lộ cánh buồm lưng; đánh trong nước nông.
  - **Voi Ma Mút Chúa**: thung lũng băng trên đảo tuyết; nằm phủ tuyết như một ụ đất, lại gần mới rùng mình đứng dậy; bão tuyết khi đánh.
  - **Carcharodontosaurus**: hẻm núi lửa, bãi xương rải quanh, nằm ăn xác trong khe đá; khe có lava.
  - **Giganotosaurus**: hẻm núi cao nguyên sa mạc, bóng đổ trên vách, vết chân to dẫn tới; bão cát chặn lối ra.
  - Mỗi trùm có màn mở đầu kèm tên, nhạc chiến đấu, hoạt động lúc không đánh; xong cả 4 thì tắt lại chế độ demo mặc định.
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

## 12. Âm thanh

- [x] **Tiếng thú thật** (`realvoice.go`, `assets/voices/SOURCES.md`): mỗi loài một giọng ghép từ bản thu thú thật (tài sản công, CC0, 2 file CC BY ghi trong `CREDITS.md`).
- [ ] **Bước chân theo mặt đất**: cỏ, cát, đá, nước, tuyết, bùn, gỗ (sàn nhà); thú to bước nặng.
- [ ] **Âm thanh nền theo vùng**: gió, chim, côn trùng ban đêm, sóng biển, mưa, suối và thác, tiếng vọng trong hang; đổi theo giờ và thời tiết.
- [ ] **Tiếng nhân vật**: thở khi chạy, kêu khi bị đánh, tiếng ăn, uống, ngáp khi ngủ dậy.
- [ ] **Tiếng giao diện**: mở túi, chế tạo xong, nhặt đồ, đặt công trình, thông báo.

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
