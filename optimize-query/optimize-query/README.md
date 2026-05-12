# Database Optimization Demo (PostgreSQL & Go)

Project này minh họa các kịch bản tối ưu hóa truy vấn Database sử dụng ngôn ngữ Go thuần (với `database/sql` và driver `pgx`), áp dụng cho PostgreSQL.

## Tính năng mô phỏng

1. **N+1 Query vs JOIN**: So sánh hiệu năng khi gọi truy vấn N lần trong vòng lặp so với việc gộp lại bằng `JOIN`.
2. **Indexing**: Phân tích Execution Plan (`EXPLAIN ANALYZE`) trước và sau khi sử dụng Index (`Seq Scan` vs `Index Scan`).
3. **Offset vs Cursor Pagination**: Sự khác biệt tốc độ giữa phân trang truyền thống `OFFSET 900000` với kỹ thuật Keyset/Cursor Pagination.
4. **Connection Pool Tuning**: Hiệu ứng thắt cổ chai ở tầng ứng dụng khi chỉ số `MaxOpenConns` của DB Driver bị cấu hình sai dưới môi trường concurrent (500 Goroutines).

## Yêu cầu cài đặt

- Golang 1.20+
- PostgreSQL Server chạy tại `localhost:5432` với tài khoản `postgres:postgres` (Hoặc có thể bật container qua file `docker-compose.yml` sẵn có: `docker compose up -d`).

## Chạy Demo

### 1. Khởi tạo dữ liệu (Seeding)

Khởi chạy lệnh sau để thiết lập schema và sử dụng Worker Pool chèn 1,000,000 bài posts tốc độ cao:

```bash
go run seed.go
```

### 2. Thực thi kịch bản (Main)

Theo như yêu cầu thiết kế `main.go`, bạn có thể truyền tham số vào để chạy riêng lẻ từng kịch bản mà không cần phải gộp tất cả vào một lần chạy.

```bash
# Xem hướng dẫn chạy các case
go run main.go

# Chạy Trường hợp 1: N+1 Query vs JOIN
go run main.go case1

# Chạy Trường hợp 2: Indexing
go run main.go case2

# Chạy Trường hợp 3: Offset vs Keyset Pagination
go run main.go case3

# Chạy Trường hợp 4: Connection Pool Tuning
go run main.go case4

# Chạy tất cả các trường hợp cùng lúc
go run main.go all
```
