create database spgw default character set utf8mb4 collate utf8mb4_0900_ai_ci;

CREATE USER 'root'@'%' IDENTIFIED BY 'changeme';
grant all on *.* to 'root'@'%';
flush privileges;
