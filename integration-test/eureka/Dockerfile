FROM maven:3.9.11-eclipse-temurin-21-alpine AS build
WORKDIR /app
COPY src /app/src
COPY pom.xml /app
RUN mvn clean package -DskipTests

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=build /app/target/eureka-1.0.0.jar /app/eureka.jar
EXPOSE 8761
ENTRYPOINT ["java", "-jar", "/app/eureka.jar"]
