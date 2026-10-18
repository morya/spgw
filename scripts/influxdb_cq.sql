-- create database peaks; use peaks;

DROP CONTINUOUS QUERY cq_room_peak_10m on peaks;
DROP CONTINUOUS QUERY cq_room_peak_1h on peaks;
DROP CONTINUOUS QUERY cq_room_peak_1d on peaks;

DROP CONTINUOUS QUERY cq_corp_peak_1m on peaks;
DROP CONTINUOUS QUERY cq_corp_peak_10m on peaks;
DROP CONTINUOUS QUERY cq_corp_peak_1h on peaks;
DROP CONTINUOUS QUERY cq_corp_peak_1d on peaks;

DROP CONTINUOUS QUERY cq_site_peak_1m on peaks;
DROP CONTINUOUS QUERY cq_site_peak_10m on peaks;
DROP CONTINUOUS QUERY cq_site_peak_1h on peaks;
DROP CONTINUOUS QUERY cq_site_peak_1d on peaks;

DROP CONTINUOUS QUERY cq_room_thumbup_1h on peaks;
DROP CONTINUOUS QUERY cq_room_thumbup_1d on peaks;

-- 房间10分钟精度统计 cq_room_peak_10m
CREATE CONTINUOUS QUERY cq_room_peak_10m ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.room_peak_10m FROM peaks.autogen.room_peak_1m GROUP BY time(10m), room_id, corp_id END
-- 房间1小时精度统计 cq_room_peak_1h
CREATE CONTINUOUS QUERY cq_room_peak_1h ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.room_peak_1h FROM peaks.autogen.room_peak_10m GROUP BY time(1h), room_id, corp_id END
-- 房间1天精度统计 cq_room_peak_1d;
CREATE CONTINUOUS QUERY cq_room_peak_1d ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.room_peak_1d FROM peaks.autogen.room_peak_1h GROUP BY time(1d, -8h), room_id, corp_id END

--- 企业统计
-- 从房间汇总corp统计值 1分钟精度 cq_corp_peak_1m
CREATE CONTINUOUS QUERY cq_corp_peak_1m ON peaks BEGIN SELECT sum(all_user) AS all_user, sum(pc_count) AS pc_count, sum(h5_count) AS h5_count, sum(android_count) AS android_count, sum(ios_count) AS ios_count INTO peaks.autogen.corp_peak_1m FROM peaks.autogen.room_peak_1m GROUP BY time(1m), corp_id END
-- team10分钟精度 cq_corp_peak_10m 
CREATE CONTINUOUS QUERY cq_corp_peak_10m ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.corp_peak_10m FROM peaks.autogen.corp_peak_1m GROUP BY time(10m), corp_id END
-- 企业1小时精度 cq_corp_peak_1h
CREATE CONTINUOUS QUERY cq_corp_peak_1h ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.corp_peak_1h FROM peaks.autogen.corp_peak_10m GROUP BY time(1h), corp_id END
-- 1day cq_corp_peak_1d
CREATE CONTINUOUS QUERY cq_corp_peak_1d ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.corp_peak_1d FROM peaks.autogen.corp_peak_1h GROUP BY time(1d, -8h), corp_id END


--- 全站统计 cq_site_peak_1m   
CREATE CONTINUOUS QUERY cq_site_peak_1m ON peaks BEGIN SELECT sum(all_user) AS all_user, sum(pc_count) AS pc_count, sum(h5_count) AS h5_count, sum(android_count) AS android_count, sum(ios_count) AS ios_count INTO peaks.autogen.site_peak_1m FROM peaks.autogen.room_peak_1m GROUP BY time(1m) END
-- cq_site_peak_10m  
CREATE CONTINUOUS QUERY cq_site_peak_10m ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.site_peak_10m FROM peaks.autogen.site_peak_1m GROUP BY time(10m) END
-- cq_site_peak_1h   
CREATE CONTINUOUS QUERY cq_site_peak_1h ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.site_peak_1h FROM peaks.autogen.site_peak_10m GROUP BY time(1h) END
-- cq_site_peak_1d
CREATE CONTINUOUS QUERY cq_site_peak_1d ON peaks BEGIN SELECT max(all_user) AS all_user INTO peaks.autogen.site_peak_1d FROM peaks.autogen.site_peak_1h GROUP BY time(1d, -8h) END

-- cq_room_thumbup_1h
CREATE CONTINUOUS QUERY cq_room_thumbup_1h ON peaks BEGIN SELECT max(thumpup) AS thumpup INTO peaks.autogen.room_thumbup_1h FROM peaks.autogen.room_thumbup_1m GROUP BY time(1h),target_uid,room_id END
-- cq_room_thumbup_1d
CREATE CONTINUOUS QUERY cq_room_thumbup_1d ON peaks BEGIN SELECT max(thumpup) AS thumpup INTO peaks.autogen.room_thumbup_1d FROM peaks.autogen.room_thumbup_1h GROUP BY time(1d, -8h),target_uid,room_id END

DROP CONTINUOUS QUERY cq_room_traffic_1m on peaks;
DROP CONTINUOUS QUERY cq_room_traffic_10m on peaks;
DROP CONTINUOUS QUERY cq_room_traffic_1h on peaks;
DROP CONTINUOUS QUERY cq_room_traffic_1d on peaks;

DROP CONTINUOUS QUERY cq_corp_traffic_1m on peaks;
DROP CONTINUOUS QUERY cq_corp_traffic_10m on peaks;
DROP CONTINUOUS QUERY cq_corp_traffic_1h on peaks;
DROP CONTINUOUS QUERY cq_corp_traffic_1d on peaks;

DROP CONTINUOUS QUERY cq_site_traffic_1m on peaks;
DROP CONTINUOUS QUERY cq_site_traffic_10m on peaks;
DROP CONTINUOUS QUERY cq_site_traffic_1h on peaks;
DROP CONTINUOUS QUERY cq_site_traffic_1d on peaks;

--- 房间流量统计
CREATE CONTINUOUS QUERY cq_room_traffic_1m ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, count(DISTINCT(user_id)) AS user_count INTO peaks.autogen.room_traffic_1m FROM peaks.autogen.room_traffic_base GROUP BY time(1m), corp_id, room_id END
-- 房间1分钟精度统计
CREATE CONTINUOUS QUERY cq_room_traffic_10m ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.room_traffic_10m FROM peaks.autogen.room_traffic_1m GROUP BY time(10m), room_id, corp_id END
-- 房间1小时精度统计
CREATE CONTINUOUS QUERY cq_room_traffic_1h ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.room_traffic_1h FROM peaks.autogen.room_traffic_10m GROUP BY time(1h), room_id, corp_id END
-- 房间1天精度统计
CREATE CONTINUOUS QUERY cq_room_traffic_1d ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.room_traffic_1d FROM peaks.autogen.room_traffic_1h GROUP BY time(1d, -8h), room_id, corp_id END

--- 企业统计
CREATE CONTINUOUS QUERY cq_corp_traffic_1m ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.corp_traffic_1m FROM peaks.autogen.room_traffic_1m GROUP BY time(1m), corp_id END
-- 企业10分钟精度
CREATE CONTINUOUS QUERY cq_corp_traffic_10m ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.corp_traffic_10m FROM peaks.autogen.corp_traffic_1m GROUP BY time(10m), corp_id END
-- 企业1小时精度
CREATE CONTINUOUS QUERY cq_corp_traffic_1h ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.corp_traffic_1h FROM peaks.autogen.corp_traffic_10m GROUP BY time(1h), corp_id END
-- 企业1天精度统计
CREATE CONTINUOUS QUERY cq_corp_traffic_1d ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.corp_traffic_1d FROM peaks.autogen.corp_traffic_1h GROUP BY time(1d, -8h), corp_id END

--- 全站统计
CREATE CONTINUOUS QUERY cq_site_traffic_1m ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.site_traffic_1m FROM peaks.autogen.room_traffic_1m GROUP BY time(1m) END
-- 全站10分钟精度
CREATE CONTINUOUS QUERY cq_site_traffic_10m ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.site_traffic_10m FROM peaks.autogen.site_traffic_1m GROUP BY time(10m) END
-- 全站1小时精度
CREATE CONTINUOUS QUERY cq_site_traffic_1h ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.site_traffic_1h FROM peaks.autogen.site_traffic_10m GROUP BY time(1h) END
-- 全站1天精度统计
CREATE CONTINUOUS QUERY cq_site_traffic_1d ON peaks BEGIN SELECT sum(traffic_bytes) AS traffic_bytes, sum(user_count) AS user_count INTO peaks.autogen.site_traffic_1d FROM peaks.autogen.site_traffic_1h GROUP BY time(1d, -8h) END


-- -- 房间10分钟精度统计 cq_room_peak_10m
-- CREATE CONTINUOUS QUERY cq_room_peak_10m ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.room_peak_10m FROM peaks.autogen.room_peak_1m GROUP BY time(10m), room_id, corp_id END
-- -- 房间1小时精度统计 cq_room_peak_1h
-- CREATE CONTINUOUS QUERY cq_room_peak_1h ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.room_peak_1h FROM peaks.autogen.room_peak_10m GROUP BY time(1h), room_id, corp_id END
-- -- 房间1天精度统计 cq_room_peak_1d;
-- CREATE CONTINUOUS QUERY cq_room_peak_1d ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.room_peak_1d FROM peaks.autogen.room_peak_1h GROUP BY time(1d, -8h), room_id, corp_id END

-- --- 企业统计
-- -- 从房间汇总corp统计值 1分钟精度 cq_corp_peak_1m
-- CREATE CONTINUOUS QUERY cq_corp_peak_1m ON peaks BEGIN SELECT sum(all_user) AS all_user, sum(pc_count) AS pc_count, sum(h5_count) AS h5_count, sum(android_count) AS android_count, sum(ios_count) AS ios_count INTO peaks.autogen.corp_peak_1m FROM peaks.autogen.room_peak_1m GROUP BY time(1m), corp_id END
-- -- team10分钟精度 cq_corp_peak_10m 
-- CREATE CONTINUOUS QUERY cq_corp_peak_10m ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.corp_peak_10m FROM peaks.autogen.corp_peak_1m GROUP BY time(10m), corp_id END
-- -- 企业1小时精度 cq_corp_peak_1h
-- CREATE CONTINUOUS QUERY cq_corp_peak_1h ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.corp_peak_1h FROM peaks.autogen.corp_peak_10m GROUP BY time(1h), corp_id END
-- -- 1day cq_corp_peak_1d
-- CREATE CONTINUOUS QUERY cq_corp_peak_1d ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.corp_peak_1d FROM peaks.autogen.corp_peak_1h GROUP BY time(1d, -8h), corp_id END


-- --- 全站统计 cq_site_peak_1m   
-- CREATE CONTINUOUS QUERY cq_site_peak_1m ON peaks BEGIN SELECT sum(all_user) AS all_user, sum(pc_count) AS pc_count, sum(h5_count) AS h5_count, sum(android_count) AS android_count, sum(ios_count) AS ios_count INTO peaks.autogen.site_peak_1m FROM peaks.autogen.room_peak_1m GROUP BY time(1m) END
-- -- cq_site_peak_10m  
-- CREATE CONTINUOUS QUERY cq_site_peak_10m ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.site_peak_10m FROM peaks.autogen.site_peak_1m GROUP BY time(10m) END
-- -- cq_site_peak_1h   
-- CREATE CONTINUOUS QUERY cq_site_peak_1h ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.site_peak_1h FROM peaks.autogen.site_peak_10m GROUP BY time(1h) END
-- -- cq_site_peak_1d
-- CREATE CONTINUOUS QUERY cq_site_peak_1d ON peaks BEGIN SELECT max(all_user) AS all_user, max(pc_count) AS pc_count, max(h5_count) AS h5_count, max(android_count) AS android_count, max(ios_count) AS ios_count INTO peaks.autogen.site_peak_1d FROM peaks.autogen.site_peak_1h GROUP BY time(1d, -8h) END

--- precision rfc3339
--- select * from corp_peak_1h where corp_id = 1005 tz('Asia/Shanghai');