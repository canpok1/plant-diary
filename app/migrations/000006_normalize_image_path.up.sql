-- diary.image_path から 'data/photos/' プレフィックスを除去して新形式に正規化する
-- 修正前: data/photos/20260216_1110_UTC.jpg または data/photos/{uuid}/20260301_100234_UTC.jpg
-- 修正後: 20260216_1110_UTC.jpg または {uuid}/20260301_100234_UTC.jpg
UPDATE diary
SET image_path = SUBSTR(image_path, LENGTH('data/photos/') + 1)
WHERE image_path LIKE 'data/photos/%';
