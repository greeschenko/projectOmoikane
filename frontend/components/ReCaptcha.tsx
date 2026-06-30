"use client";

import { useEffect, useRef } from "react";
import ReCAPTCHA from "react-google-recaptcha";

interface ReCaptchaProps {
  onChange: (token: string | null) => void;
}

export default function ReCaptcha({ onChange }: ReCaptchaProps) {
  const siteKey = process.env.NEXT_PUBLIC_RECAPTCHA_SITE_KEY;
  const recaptchaRef = useRef<ReCAPTCHA>(null);

  useEffect(() => {
    return () => {
      recaptchaRef.current?.reset();
    };
  }, []);

  if (!siteKey) return null;

  return (
    <ReCAPTCHA
      ref={recaptchaRef}
      sitekey={siteKey}
      onChange={onChange}
      sx={{ mb: 2 }}
    />
  );
}
