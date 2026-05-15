import { Languages } from "lucide-react";
import { useI18n } from "../context/I18nContext";

export function LanguageToggle() {
  const { toggleLanguage, t } = useI18n();
  return (
    <button className="icon-button secondary" type="button" onClick={toggleLanguage} title="Switch language">
      <Languages size={18} aria-hidden="true" />
      <span>{t("language.toggle")}</span>
    </button>
  );
}
