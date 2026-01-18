"use client";

import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { LangToggle } from "@/components/lang-toggle";

export default function ListPage() {
  const [lang, setLang] = useState<"zh" | "en">("zh");
  const t = useMemo(() => messages[lang], [lang]);
  const [images, setImages] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  const fetchImages = async (query = "") => {
    try {
      setLoading(true);
      const res = await fetch(`/api/list?q=${query}`);
      if (res.ok) {
        const data = await res.json();
        setImages(data.images || []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      fetchImages(search);
    }, 500);
    return () => clearTimeout(timer);
  }, [search]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold tracking-tight">{t.title}</h2>
          <LangToggle value={lang} onChange={setLang} />
        </div>
        <Input
          className="max-w-sm"
          placeholder={t.search}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {loading && <p>{t.loading}</p>}

      {!loading && images.length === 0 && (
         <div className="text-center py-10 text-muted-foreground">{t.empty}</div>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {images.map((img) => (
          <Card key={img}>
            <CardContent className="p-6 flex flex-col justify-between space-y-4">
              <div className="space-y-1">
                <h3 className="font-semibold text-lg break-all">{img}</h3>
                <p className="text-sm text-muted-foreground">{t.stored}</p>
              </div>
              <div className="flex gap-2">
                  <Button asChild className="w-full">
                    <a href={`/api/files/${img}`} download>{t.download}</a>
                  </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

const messages = {
  zh: {
    title: "已下载镜像",
    search: "搜索镜像...",
    loading: "加载中...",
    empty: "暂无镜像",
    stored: "已存储本地",
    download: "下载 tar",
  },
  en: {
    title: "Downloaded Images",
    search: "Search images...",
    loading: "Loading...",
    empty: "No images found.",
    stored: "Stored locally",
    download: "Download tar",
  },
};
