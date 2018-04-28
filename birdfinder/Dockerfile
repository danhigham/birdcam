FROM golang:latest

RUN apt-get update && \
  apt-get upgrade -y && \
  apt-get install -y build-essential unzip build-essential cmake git libgtk2.0-dev pkg-config libavcodec-dev libavformat-dev libswscale-dev libtbb2 libtbb-dev libjpeg-dev libpng-dev libtiff-dev libdc1394-22-dev

RUN mkdir /tmp/opencv && \
  cd /tmp/opencv && \
  wget -O opencv.zip https://github.com/opencv/opencv/archive/3.4.1.zip && \
  unzip opencv.zip && \
  wget -O opencv_contrib.zip https://github.com/opencv/opencv_contrib/archive/3.4.1.zip && \
  unzip opencv_contrib.zip

RUN cd /tmp/opencv/opencv-3.4.1 && \
  mkdir build && \
  cd build && \
	cmake -D CMAKE_BUILD_TYPE=RELEASE -D CMAKE_INSTALL_PREFIX=/usr/local -D OPENCV_EXTRA_MODULES_PATH=/tmp/opencv/opencv_contrib-3.4.1/modules -D BUILD_DOCS=OFF BUILD_EXAMPLES=OFF -D BUILD_TESTS=OFF -D BUILD_PERF_TESTS=OFF -D BUILD_opencv_java=OFF -D BUILD_opencv_python=OFF -D BUILD_opencv_python2=OFF -D BUILD_opencv_python3=OFF .. && \
	make -j4 && \
	make install && \
	ldconfig

RUN cd ~ && \
  rm -rf /tmp/opencv

RUN go get github.com/danhigham/birdcam/birdfinder && \
  cd $GOPATH/src/github.com/danhigham/birdcam/birdfinder && \
  go get .

