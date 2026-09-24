// Minimal ambient declaration for `react-google-recaptcha` (the package ships
// no types and @types is not vendored). Only the surface used by the app:
// ReCaptcha.tsx (ref + reset, sitekey, onChange) and the `sx` prop it passes
// through. Dev mode never type-checks this; the prod image does.
declare module "react-google-recaptcha" {
  import * as React from "react";

  export interface ReCAPTCHAProps {
    sitekey: string;
    onChange?: (token: string | null) => void;
    size?: "normal" | "compact" | "invisible";
    theme?: "light" | "dark";
    hl?: string;
    tabindex?: number;
    sx?: Record<string, unknown>;
  }

  export default class ReCAPTCHA extends React.Component<ReCAPTCHAProps> {
    reset(): void;
    execute(): void;
  }
}