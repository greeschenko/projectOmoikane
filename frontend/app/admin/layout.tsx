import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";
import AdminLayout from "@/components/AdminLayout";

export default async function AdminRootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await getSession();
  if (!session || session.role !== "admin") {
    redirect("/login");
  }
  return <AdminLayout session={session}>{children}</AdminLayout>;
}
