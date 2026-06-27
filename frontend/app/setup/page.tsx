import { redirect } from "next/navigation";
import { apiFetch } from "@/lib/api";
import SetupForm from "@/components/SetupForm";

export const dynamic = "force-dynamic";

export default async function SetupPage() {
  const data = await apiFetch<{ setupRequired: boolean }>("/setup/check");
  if (!data.setupRequired) {
    redirect("/login");
  }
  return <SetupForm />;
}
