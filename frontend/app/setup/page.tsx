import { redirect } from "next/navigation";
import store from "@/lib/store";
import SetupForm from "@/components/SetupForm";

export const dynamic = "force-dynamic";

export default function SetupPage() {
  if (store.userCount > 0) {
    redirect("/login");
  }
  return <SetupForm />;
}
