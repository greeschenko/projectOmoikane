import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAdmin } from "@/lib/auth";

export async function GET() {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const settings = store.getSettings();
  return NextResponse.json(settings);
}

export async function PUT(request: Request) {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await request.json();
  const { siteName, tagline, logo, favicon, blogEnabled } = body;
  const settings = store.updateSettings({
    ...(siteName !== undefined && { siteName }),
    ...(tagline !== undefined && { tagline }),
    ...(logo !== undefined && { logo }),
    ...(favicon !== undefined && { favicon }),
    ...(blogEnabled !== undefined && { blogEnabled }),
  });
  return NextResponse.json(settings);
}
