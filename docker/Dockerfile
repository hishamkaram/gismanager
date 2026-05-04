# GeoServer dev/test image used by gismanager's integration suite.
#
# Base: Tomcat 9 + JDK 17 (Temurin). GeoServer 2.27 LTS / 2.28 stable use the
# javax.* servlet namespace (Servlet 4.x) and do not run on Tomcat 10/11
# (which moved to jakarta.*). GeoServer 3.0 (Tomcat 11 / Jakarta EE) is
# tracked separately for a future v1.x point release.
#
# This image is purposely minimal — gismanager's integration tests exercise
# only the publish flow (workspace + PostGIS datastore + feature type). No
# Importer / Monitor extensions are required (the lower-level geoserver/v2
# client's integration suite has those; gismanager does not).

ARG TOMCAT_TAG=9-jdk17-temurin
FROM tomcat:${TOMCAT_TAG}

LABEL maintainer="Hesham Karm<hishamwaleedkaram@gmail.com>"

ARG GEOSERVER_VERSION=2.28.0
ARG GEOSERVER_DOWNLOAD_URL=https://downloads.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/geoserver-${GEOSERVER_VERSION}-war.zip

ENV DEBIAN_FRONTEND=noninteractive \
    GEOSERVER_VERSION=${GEOSERVER_VERSION} \
    JAVA_OPTS="-Xms512m -Xmx2g -XX:+UseG1GC -Djava.awt.headless=true"

# No apt-install needed: the Tomcat base image already ships curl +
# ca-certificates, and the JDK's `jar` tool can extract the WAR archive
# (replacing `unzip`). Skipping apt entirely removes the most common
# transient-failure mode on GHA — the 2.27.4 leg of integration #7 lost
# 19 minutes to `apt-get update` exit 100 from an Ubuntu mirror.
RUN set -eux; \
    rm -rf /usr/local/tomcat/webapps/*; \
    cd /tmp; \
    curl -fsSL --proto '=https' --tlsv1.2 \
         --retry 5 --retry-delay 10 --retry-all-errors \
         -o geoserver.war.zip "${GEOSERVER_DOWNLOAD_URL}"; \
    mkdir -p /tmp/geoserver; \
    cd /tmp/geoserver; \
    jar -xf /tmp/geoserver.war.zip; \
    mv /tmp/geoserver/geoserver.war /usr/local/tomcat/webapps/geoserver.war; \
    rm -rf /tmp/geoserver /tmp/geoserver.war.zip

# Pre-extract the WAR. Tomcat will run the unpacked form just like the WAR
# but starts faster on subsequent runs. `jar -xf` again — no apt-installed
# unzip needed.
RUN set -eux; \
    cd /usr/local/tomcat/webapps; \
    mkdir geoserver; \
    cd geoserver; \
    jar -xf ../geoserver.war; \
    rm ../geoserver.war

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=5 \
    CMD curl -fsS http://localhost:8080/geoserver/web/ || exit 1
