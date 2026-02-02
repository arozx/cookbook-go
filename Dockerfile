FROM alpine:latest

WORKDIR /app

# 1. Copy your specific binary from your dist folder to the container
# We rename it to 'server' inside the container for simplicity
COPY recipe-tracker ./server

# 2. Make sure it's executable
RUN chmod +x ./server

# 3. (Optional) If your app uses an external config file or .env, copy it too:
# COPY config.yaml .
# COPY .env .

# 4. Expose the port your app uses (Check your code! I'm assuming 8080)
EXPOSE 9005

# 5. Run it
CMD ["./server"]
