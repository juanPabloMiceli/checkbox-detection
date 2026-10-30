FROM golang:1.27.3-bookworm

# GoCV needs OpenCV 4.13.0. Compile it once as part of the image.
RUN apt-get update && apt-get install -y --no-install-recommends \
    cmake pkg-config libjpeg62-turbo-dev libpng-dev curl \
    && rm -rf /var/lib/apt/lists/*
RUN git clone --depth 1 --branch 4.13.0 https://github.com/opencv/opencv.git /tmp/opencv \
    && test "$(git -C /tmp/opencv rev-parse HEAD)" = fe38fc608f6acb8b68953438a62305d8318f4fcd \
    && cmake -S /tmp/opencv -B /tmp/opencv-build \
        -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local \
        -DOPENCV_GENERATE_PKGCONFIG=ON -DBUILD_TESTS=OFF -DBUILD_PERF_TESTS=OFF \
        -DBUILD_EXAMPLES=OFF -DBUILD_opencv_gapi=OFF -DWITH_IPP=OFF \
    && cmake --build /tmp/opencv-build --parallel 2 \
    && cmake --install /tmp/opencv-build \
    && ldconfig \
    && rm -rf /tmp/opencv /tmp/opencv-build

# Keep source and tools available so reviewers can also run tests in the image.
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go main_test.go ./
COPY internal/ internal/
COPY testdata/ testdata/
COPY compose.yaml config.example.json /deploy/
RUN go build -mod=readonly -o /usr/local/bin/checkbox-server .

# Run the application without root privileges.
RUN useradd --uid 10001 --create-home checkbox
USER checkbox
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD curl --fail --silent --show-error --max-time 2 http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["checkbox-server"]
CMD ["-addr", "0.0.0.0:8080"]
