import { redirect } from "next/navigation";
import { apiFetch } from "@/lib/api";
import HomePage from "@/components/HomePage";

export const dynamic = "force-dynamic";

export default async function Home() {
  const data = await apiFetch<{ setupRequired: boolean }>("/setup/check");
  if (data.setupRequired) {
    redirect("/setup");
  }
  return <HomePage />;
}
