-- diary.image_path に 'data/photos/' プレフィックスを付与して旧形式に戻す
-- NOT LIKE 条件により、すでにプレフィックスが付いている行への二重付与を防ぐ
-- 注意: migration 006 適用前から新形式で保存されていた行にも誤ってプレフィックスが付与される可能性がある
UPDATE diary
SET image_path = 'data/photos/' || image_path
WHERE image_path NOT LIKE 'data/photos/%';
