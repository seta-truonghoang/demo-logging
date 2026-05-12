# 🐳 Demo Docker Tối Ưu Cho 3 Ngôn Ngữ: Go, Node.js & Java

Chào mừng bạn đến với bộ demo Docker mẫu mực cho 3 ngôn ngữ phổ biến: **Go**, **Node.js** và **Java (phiên bản 21 sử dụng Virtual Threads)**. Mỗi dịch vụ chạy một HTTP server đơn giản trả về siêu dữ liệu (metadata) của môi trường, nhưng phần **Dockerfile** được viết và thiết kế cực kỳ cẩn thận, tuân thủ nghiêm ngặt các tiêu chuẩn **Production-Ready** tiên tiến nhất.

---

## 🛠️ Cấu Trúc Thư Mục Dự Án

Toàn bộ mã nguồn đã được tổ chức quy củ tại thư mục `docker-demo/`:

*   [docker-compose.yml](./docker-compose.yml) — Định nghĩa phối hợp cả 3 dịch vụ và cấu hình giới hạn tài nguyên.
*   **Go Service (`golang/`)**
    *   [go.mod](./golang/go.mod) — Module Go.
    *   [main.go](./golang/main.go) — API server bằng thư viện chuẩn Go.
    *   [Dockerfile](./golang/Dockerfile) — Dockerfile tối ưu hóa tối đa kích thước.
*   **Node.js Service (`nodejs/`)**
    *   [package.json](./nodejs/package.json) & [package-lock.json](./nodejs/package-lock.json) — Quản lý dependencies.
    *   [server.js](./nodejs/server.js) — HTTP server Node.js hỗ trợ Graceful Shutdown.
    *   [Dockerfile](./nodejs/Dockerfile) — Dockerfile bảo mật tích hợp `tini`.
*   **Java Service (`java/`)**
    *   [pom.xml](./java/pom.xml) — Trình quản lý Maven.
    *   [App.java](./java/src/main/java/com/demo/App.java) — HTTP server siêu nhẹ sử dụng **Java 21 Virtual Threads**.
    *   [Dockerfile](./java/Dockerfile) — Dockerfile tối ưu hoá Maven cache và kích thước JRE.

---

## 🚀 Các Triết Lý Thiết Kế Dockerfile Đỉnh Cao Được Áp Dụng

Dưới đây là phân tích chi tiết về tính "cẩn thận" và các kỹ thuật nâng cao được đưa vào từng Dockerfile:

### 1. Multi-Stage Builds (Biên dịch đa giai đoạn)
*   **Cách thức**: Tách biệt hoàn toàn quá trình biên dịch (cần nhiều SDK, công cụ biên dịch nặng nề) và quá trình chạy (chỉ cần môi trường runtime tối thiểu).
*   **Lợi ích**: Giảm kích thước image từ hàng trăm MB xuống chỉ còn vài chục MB, đồng thời loại bỏ mã nguồn khỏi môi trường chạy để bảo vệ tài sản trí tuệ và thu hẹp bề mặt tấn công bảo mật (attack surface).

### 2. Dependency Layer Caching (Tận dụng Docker Cache cho thư viện)
*   **Cách thức**: Luôn copy các tệp mô tả thư viện (`go.mod`, `package*.json`, `pom.xml`) trước, sau đó chạy lệnh tải thư viện (`go mod download`, `npm ci`, `mvn dependency:go-offline`), rồi mới sao chép mã nguồn chính (`COPY src` / `COPY .`).
*   **Lợi ích**: Khi bạn chỉ thay đổi mã nguồn mà không thêm thư viện mới, Docker sẽ bỏ qua bước tải thư viện (bước tốn thời gian nhất), giúp tốc độ rebuild giảm từ vài phút xuống còn **vài giây**.

### 3. Non-Root Execution (Chạy dưới quyền hạn thấp)
*   **Cách thức**: Mặc định Docker chạy tiến trình bằng user `root`. Nếu hacker chiếm quyền kiểm soát ứng dụng, chúng sẽ kiểm soát cả máy chủ vật lý. Trong cả 3 Dockerfile, chúng ta tạo ra group và user riêng (`appuser` / `node`) và gọi lệnh `USER appuser` trước khi chạy.
*   **Lợi ích**: Bảo vệ hệ thống máy chủ, ngăn chặn tuyệt đối các lỗ hổng leo thang đặc quyền (Privilege Escalation).

### 4. Tín Hiệu Hệ Thống & PID 1 (Graceful Shutdown)
*   **Cách thức**:
    *   **Node.js**: Mặc định tiến trình Node.js không xử lý tốt các tín hiệu của hệ điều hành (như `SIGTERM` từ lệnh `docker stop`) nếu chạy ở vị trí PID 1. Chúng ta cài đặt `tini` làm tiến trình init (PID 1) để dọn dẹp các tiến trình con "zombie" và chuyển tiếp tín hiệu tắt chuẩn xác.
    *   **Java**: Sử dụng lệnh `exec java ...` trong entrypoint dạng shell để tiến trình JVM thay thế trực tiếp shell và nhận trực tiếp các tín hiệu hệ điều hành mà không bị kẹt bởi shell gia.
*   **Lợi ích**: Cho phép container tắt êm ái (Graceful Shutdown), hoàn thành nốt các request đang xử lý dở dang trước khi dừng hoàn toàn.

---

## 🔍 Điểm Nhấn Tối Ưu Riêng Cho Từng Ngôn Ngữ

### 🐹 Golang
*   **CGO_ENABLED=0**: Biên dịch ứng dụng thành **Static Binary** độc lập hoàn toàn, không phụ thuộc vào bất kỳ thư viện C động nào của hệ điều hành (libc).
*   **-ldflags="-s -w"**: Loại bỏ toàn bộ bảng ký hiệu debug (debug symbols) và thông tin DWARF, giúp giảm kích thước file thực thi khoảng 50%-70% mà không ảnh hưởng hiệu năng.
*   Sử dụng base image **Alpine** siêu nhỏ giúp tổng kích thước image chỉ rơi vào khoảng **~15MB**!

### 🟢 Node.js
*   **npm ci (Clean Install)**: Cài đặt thư viện dựa vào `package-lock.json` một cách nghiêm ngặt, nhanh hơn nhiều so với `npm install` truyền thống và tránh hiện tượng sai lệch phiên bản giữa môi trường dev và production.
*   **NODE_ENV=production**: Tự động bật các chế độ tối ưu hóa hiệu năng và tắt log debug của các framework/thư viện bên thứ ba.

### ☕ Java (Virtual Threads + JRE)
*   **Java 21 Virtual Threads**: Sử dụng luồng ảo giúp HTTP server xử lý hàng trăm nghìn kết nối đồng thời với lượng RAM cực ít, khắc phục hoàn toàn điểm yếu tốn tài nguyên truyền thống của Java.
*   **Eclipse Temurin JRE-Alpine**: Chỉ sử dụng JRE (Java Runtime Environment) thay vì JDK đầy đủ để chạy ứng dụng, giúp tiết kiệm hàng trăm MB dung lượng ổ đĩa.
*   **JVM Tuning trong Container**: Cấu hình các tham số `-XX:+UseG1GC` (bộ gom rác tối ưu) và `-XX:+ExitOnOutOfMemoryError` (buộc container sập ngay khi tràn bộ nhớ để hệ thống như Kubernetes tự động khởi động lại container sạch mới).

---

## 🚀 Hướng Dẫn Chạy Và Kiểm Tra

Bạn có thể chạy toàn bộ 3 dịch vụ đồng thời bằng cách sử dụng Docker Compose.

### 1. Khởi động các dịch vụ
Di chuyển vào thư mục `docker-demo` và khởi động biên dịch + chạy:
```bash
cd docker-demo
docker compose up -d --build
```

### 2. Kiểm tra trạng thái hoạt động
```bash
docker compose ps
```

### 3. Kiểm tra API của từng dịch vụ
Sử dụng `curl` để gọi API từ các cổng tương ứng:

*   **Golang (Port 8080):**
    ```bash
    curl http://localhost:8080
    ```
    *Phản hồi mẫu:*
    ```json
    {"language":"Go","message":"Xin chào từ Docker Container tối ưu cho Go!","timestamp":"2026-05-13T05:10:00Z","hostname":"demo-golang-service","version":"1.0.0"}
    ```

*   **Node.js (Port 8081):**
    ```bash
    curl http://localhost:8081
    ```
    *Phản hồi mẫu:*
    ```json
    {"language":"Node.js","message":"Xin chào từ Docker Container tối ưu cho Node.js!","timestamp":"2026-05-13T05:10:00.000Z","hostname":"demo-nodejs-service","version":"1.0.0"}
    ```

*   **Java (Port 8082):**
    ```bash
    curl http://localhost:8082
    ```
    *Phản hồi mẫu:*
    ```json
    {"language":"Java","message":"Xin chào từ Docker Container tối ưu cho Java (Virtual Threads)!","timestamp":"2026-05-13T05:10:00Z","hostname":"demo-java-service","version":"1.0.0"}
    ```

### 4. Dọn dẹp tài nguyên
Khi muốn tắt và xóa bỏ toàn bộ container:
```bash
docker compose down
```
