# Base Image for building artifact. Bundled with maven
FROM openjdk:8-jdk-buster as builder
COPY . /src/
WORKDIR /src/
RUN apt-get update && apt-get install -y maven && apt-get clean
RUN mvn -B clean package -f pom.xml -X -e -DskipTests

FROM openjdk:8-jre-buster as baseruntime

RUN apt-get update && apt-get install -y curl dnsutils netcat vim less procps telnet && apt-get clean

RUN mkdir -p /opt/hadoop
RUN mkdir -p /opt/hbase

COPY --from=builder [ "/src/external_lib/hadoop-*", "/opt/hadoop" ]
COPY --from=builder [ "/src/external_lib/hbase-*", "/opt/hbase" ]
COPY --from=builder [ "/src/utilities/scripts", "/opt/scripts" ]

RUN rm -r /opt/hadoop/share/doc

RUN addgroup --gid 1011 hbase
RUN useradd --create-home --uid 1011 --gid 1011 --shell /bin/bash --system hbase

RUN mkdir -p /var/log/hbase /etc/hbase /var/run/hbase
RUN mkdir -p /var/log/hadoop /etc/hadoop /var/run/hadoop
RUN chown -R hbase:hbase /var/log/hbase /etc/hbase /var/run/hbase 
RUN chown -R hbase:hbase /var/log/hadoop /etc/hadoop /var/run/hadoop
