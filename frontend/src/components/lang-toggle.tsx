"use client";

import { Button } from "@/components/ui/button";

type Props = {
  value: "zh" | "en";
  onChange: (lang: "zh" | "en") => void;
};

export function LangToggle({ value, onChange }: Props) {
  return (
    <div className="inline-flex items-center gap-2">
      <Button
        variant={value === "zh" ? "default" : "outline"}
        className="h-9 px-3"
        onClick={() => onChange("zh")}
      >
        中文
      </Button>
      <Button
        variant={value === "en" ? "default" : "outline"}
        className="h-9 px-3"
        onClick={() => onChange("en")}
      >
        EN
      </Button>
    </div>
  );
}
