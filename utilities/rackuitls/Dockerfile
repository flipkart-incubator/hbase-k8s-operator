FROM openjdk:8-jdk-buster as builder
ARG AppName 
COPY . /src/
WORKDIR /src/
RUN apt-get update && apt-get -y install maven && apt-get clean
RUN mvn -U -DskipTests=true install

FROM openjdk:8-jre-buster as baseruntime
ARG AppName

RUN apt-get update && apt-get install -y dnsutils netcat vim less procps curl && apt-get clean

COPY --from=builder [ "/src/target/classes/logback.xml", "/etc/${AppName}/" ]
COPY --from=builder [ "/src/target/", "/var/lib/${AppName}/" ]
COPY --from=builder [ "/src/entrypoint", "/entrypoint" ]

RUN addgroup --gid 1011 hbase
RUN useradd --create-home --uid 1011 --gid 1011 --shell /bin/bash --system hbase

RUN mkdir /var/log/flipkart && chown hbase:hbase /var/log/flipkart

RUN chmod 777 /entrypoint

RUN sed -i "s/__PACKAGE__/$AppName/g" "/entrypoint"
RUN mkdir -p "/etc/${AppName}"

RUN chmod +x "/entrypoint"
ENTRYPOINT [ "/entrypoint" ]
CMD ["com.flipkart.hbase.HbaseRackUtils", "/etc/hbase", "/hbase-operator", "/tmp/rack_topology.data"]
