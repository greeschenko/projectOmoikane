import { cookies } from "next/headers";
import store from "./store";

const SESSION_COOKIE = "session";

export interface Session {
  userId: string;
  role: "admin" | "user";
}

export async function getSession(): Promise<Session | null> {
  const cookieStore = await cookies();
  const value = cookieStore.get(SESSION_COOKIE)?.value;
  if (!value) return null;
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

export async function setSession(userId: string, role: string): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(
    SESSION_COOKIE,
    JSON.stringify({ userId, role }),
    { httpOnly: true, secure: false, sameSite: "lax", path: "/", maxAge: 86400 }
  );
}

export async function clearSession(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(SESSION_COOKIE);
}

export function requireAdmin(session: Session | null): void {
  if (!session || session.role !== "admin") {
    throw new Error("Unauthorized");
  }
}

export function requireAuth(session: Session | null): void {
  if (!session) throw new Error("Unauthorized");
}
