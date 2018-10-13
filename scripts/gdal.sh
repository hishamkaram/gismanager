#!/bin/bash
sudo apt-get purge --auto-remove gdal-bin
sudo apt install build-essential python-all-dev 
mkdir ./go_gdal && cd ./go_gdal && wget http://download.osgeo.org/gdal/2.3.2/gdal232.zip && \
unzip gdal232.zip && cd gdal-2.3.2
./configure --prefix=/usr/
make
sudo make install
cd ../..