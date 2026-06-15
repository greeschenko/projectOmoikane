import { redirect } from "next/navigation";
import store from "@/lib/store";
import HomePage from "@/components/HomePage";

export default function Home() {
  if (store.userCount === 0) {
    redirect("/setup");
  }
  return <HomePage />;
}
