"use client";

import { useState, useMemo } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { LangToggle } from "@/components/lang-toggle";

export default function Home() {
  const router = useRouter();
  const [lang, setLang] = useState<"zh" | "en">("zh");
  const t = useMemo(() => messages[lang], [lang]);
  const [loading, setLoading] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [formData, setFormData] = useState({
    image: "",
    platform: "linux/amd64",
    registry: "",
    username: "",
    password: "",
  });
  const [error, setError] = useState("");
  const [logs, setLogs] = useState<string[]>([]);

  const addLog = (msg: string) => setLogs((prev) => [...prev, msg]);

  const toggleAdvanced = () => {
    setShowAdvanced((prev) => {
      if (prev) {
        setFormData((data) => ({
          ...data,
          registry: "",
          username: "",
          password: "",
        }));
      }
      return !prev;
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    setLogs([]);

    addLog(
      `${t.starting} ${formData.image} (${formData.platform})...`
    );

    try {
      const res = await fetch("/api/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || t.failed);
      }

      addLog(t.successRedirect);
      setTimeout(() => {
        router.push("/list");
      }, 1000);
    } catch (err: any) {
      setError(err.message);
      addLog(`${t.errorPrefix} ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-10">
      <div className="flex flex-col items-center text-center gap-4">
        <div className="flex w-full items-center justify-between">
          <div className="flex-1" />
          <LangToggle value={lang} onChange={setLang} />
        </div>
        <div className="space-y-3 max-w-3xl">
          <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
            {t.heroTitle}
          </h1>
          <p className="text-lg text-muted-foreground leading-relaxed">
            {t.heroDesc}
          </p>
        </div>
      </div>

      <Card className="w-full max-w-3xl mx-auto">
        <CardHeader>
          <CardTitle>{t.formTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t.imageLabel}</label>
              <Input
                placeholder="e.g. nginx:latest or myrepo/image:tag"
                value={formData.image}
                onChange={(e) =>
                  setFormData({ ...formData, image: e.target.value })
                }
                required
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t.platformLabel}</label>
                <select
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  value={formData.platform}
                  onChange={(e) =>
                    setFormData({ ...formData, platform: e.target.value })
                  }
                >
                  <option value="linux/amd64">Linux / x86_64</option>
                  <option value="linux/arm64">Linux / ARM64</option>
                  <option value="linux/arm/v7">Linux / ARMv7</option>
                </select>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">{t.registryLabel}</label>
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{t.advancedHint}</span>
                  <button
                    type="button"
                    className="text-primary underline"
                    onClick={toggleAdvanced}
                  >
                    {showAdvanced ? t.hideAdvanced : t.showAdvanced}
                  </button>
                </div>
              </div>
            </div>

            {showAdvanced && (
              <div className="space-y-4 rounded-md border bg-muted/30 p-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">{t.registryLabel}</label>
                  <Input
                    placeholder="docker.io"
                    value={formData.registry}
                    onChange={(e) =>
                      setFormData({ ...formData, registry: e.target.value })
                    }
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t.username}</label>
                    <Input
                      value={formData.username}
                      onChange={(e) =>
                        setFormData({ ...formData, username: e.target.value })
                      }
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t.password}</label>
                    <Input
                      type="password"
                      value={formData.password}
                      onChange={(e) =>
                        setFormData({ ...formData, password: e.target.value })
                      }
                    />
                  </div>
                </div>
              </div>
            )}

            {error && <p className="text-sm text-red-500">{error}</p>}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? t.loading : t.submit}
            </Button>
          </form>
        </CardContent>
      </Card>

      {logs.length > 0 && (
        <Card className="w-full max-w-3xl mx-auto bg-slate-950 text-slate-50">
           <CardContent className="p-4 font-mono text-xs space-y-1">
              {logs.map((log, i) => <div key={i}>{'>'} {log}</div>)}
           </CardContent>
        </Card>
      )}
    </div>
  );
}

const messages = {
  zh: {
    heroTitle: "一站式镜像加速下载",
    heroDesc: "指定平台、仓库和认证信息，排队下载，自动去重，下载完成可直接列表查看或复用。",
    formTitle: "下载镜像",
    imageLabel: "镜像名称",
    platformLabel: "平台",
    registryLabel: "镜像仓库 (高级)",
    username: "用户名",
    password: "密码",
    showAdvanced: "展开高级参数",
    hideAdvanced: "收起高级参数",
    advancedHint: "默认 docker.io，需认证再填写",
    loading: "处理中...",
    submit: "开始下载",
    starting: "开始下载",
    failed: "下载失败",
    successRedirect: "下载完成，跳转列表...",
    errorPrefix: "错误",
  },
  en: {
    heroTitle: "Accelerated Docker image downloads",
    heroDesc: "Specify platform, registry, and credentials. Requests are queued, deduped, and reusable once downloaded.",
    formTitle: "Download Image",
    imageLabel: "Image name",
    platformLabel: "Platform",
    registryLabel: "Registry (advanced)",
    username: "Username",
    password: "Password",
    showAdvanced: "Show advanced",
    hideAdvanced: "Hide advanced",
    advancedHint: "Default docker.io; fill in only if needed",
    loading: "Processing...",
    submit: "Download",
    starting: "Starting download for",
    failed: "Download failed",
    successRedirect: "Download complete, redirecting to list...",
    errorPrefix: "Error",
  },
};
