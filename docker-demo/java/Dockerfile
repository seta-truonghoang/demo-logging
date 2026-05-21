# ==========================================
# STAGE 1: Build Environment
# ==========================================
FROM maven:3.9.6-eclipse-temurin-21-alpine AS builder

# Set the working directory inside the container
WORKDIR /build

# Copy pom.xml first to download dependencies/plugins and leverage Docker layer cache
COPY pom.xml .

# Download dependencies/plugins offline
RUN mvn dependency:go-offline -B

# Copy the actual application source code
COPY src ./src

# Package the application jar (skip tests for faster build speed)
RUN mvn clean package -DskipTests

# ==========================================
# STAGE 2: Runtime Environment
# ==========================================
# Use the lightweight Eclipse Temurin JRE on Alpine for optimized size and security
FROM eclipse-temurin:21-jre-alpine

# Create a non-privileged group and user for secure runtime execution
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the built JAR from the builder stage
COPY --from=builder /build/target/java-docker-demo-1.0.0.jar /app/app.jar

# Change ownership of the folder to the non-root user
RUN chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

# Configure default environment variables for JVM tuning:
# -XX:+UseG1GC: Uses G1 Garbage Collector optimized for container environments
# -XX:+ExitOnOutOfMemoryError: Forces container to crash immediately on OOM so orchestrators can restart a healthy container
ENV PORT=8082 \
    JAVA_OPTS="-XX:+UseG1GC -XX:+ExitOnOutOfMemoryError"

# Expose port
EXPOSE 8080

# Run the Java application
# We use 'exec' to ensure JVM receives OS shutdown signals directly as PID 1
ENTRYPOINT ["sh", "-c", "exec java $JAVA_OPTS -jar /app/app.jar"]
