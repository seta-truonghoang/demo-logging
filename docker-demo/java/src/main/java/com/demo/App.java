package com.demo;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.InetAddress;
import java.time.Instant;
import java.util.concurrent.Executors;

public class App {

    public static void main(String[] args) throws IOException {
        String portStr = System.getenv("PORT");
        int port = (portStr != null) ? Integer.parseInt(portStr) : 8082;

        String hostname;
        try {
            hostname = InetAddress.getLocalHost().getHostName();
        } catch (Exception e) {
            hostname = "unknown";
        }

        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        
        // Use Virtual Threads (Java 21) for ultra-high performance concurrency with minimal resource usage
        server.setExecutor(Executors.newVirtualThreadPerTaskExecutor());

        String finalHostname = hostname;
        server.createContext("/", new HttpHandler() {
            @Override
            public void handle(HttpExchange exchange) throws IOException {
                String requestMethod = exchange.getRequestMethod();
                String requestPath = exchange.getRequestURI().getPath();
                System.out.println("[" + Instant.now() + "] " + requestMethod + " " + requestPath);

                if ("/".equals(requestPath) || "/api/info".equals(requestPath)) {
                    String jsonResponse = String.format(
                        "{\"language\":\"Java\",\"message\":\"Xin chào từ Docker Container tối ưu cho Java (Virtual Threads)!\",\"timestamp\":\"%s\",\"hostname\":\"%s\",\"version\":\"1.0.0\"}",
                        Instant.now().toString(),
                        finalHostname
                    );

                    byte[] responseBytes = jsonResponse.getBytes("UTF-8");
                    exchange.getResponseHeaders().set("Content-Type", "application/json; charset=UTF-8");
                    exchange.sendResponseHeaders(200, responseBytes.length);
                    try (OutputStream os = exchange.getResponseBody()) {
                        os.write(responseBytes);
                    }
                } else {
                    String jsonResponse = "{\"error\":\"Not Found\"}";
                    byte[] responseBytes = jsonResponse.getBytes("UTF-8");
                    exchange.getResponseHeaders().set("Content-Type", "application/json; charset=UTF-8");
                    exchange.sendResponseHeaders(404, responseBytes.length);
                    try (OutputStream os = exchange.getResponseBody()) {
                        os.write(responseBytes);
                    }
                }
            }
        });

        System.out.println("Java application (Virtual Threads) is starting on port " + port + "...");
        server.start();
    }
}
