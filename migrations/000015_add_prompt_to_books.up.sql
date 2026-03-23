-- books テーブルに prompt カラムを追加する
ALTER TABLE books ADD COLUMN prompt TEXT NOT NULL DEFAULT '{{book_name}}の写真を見て、成長の様子や変化を観察してください。親しみやすい口調で、200文字程度の観察日記を書いてください。
現在は{{datetime}}です。

{{past_diaries}}

これまでの観察記録を踏まえて、今回の写真から見られる成長の変化や特徴を記述してください。';
