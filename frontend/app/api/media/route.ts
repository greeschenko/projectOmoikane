import { NextResponse } from "next/server";
import { getSession, requireAdmin } from "@/lib/auth";
import store from "@/lib/store";

export async function GET() {
  try {
    const session = await getSession();
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const media = store.getMedia();
  return NextResponse.json({ media });
}

export async function POST(request: Request) {
  try {
    const session = await getSession();
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const formData = await request.formData();
    const file = formData.get("file") as File | null;
    if (!file) {
      return NextResponse.json({ error: "No file provided" }, { status: 400 });
    }

    const MAX_SIZE = 10 * 1024 * 1024;
    if (file.size > MAX_SIZE) {
      return NextResponse.json({ error: "File too large (max 10MB)" }, { status: 400 });
    }

    const allowedTypes = ["image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml"];
    if (!allowedTypes.includes(file.type)) {
      return NextResponse.json({ error: "Invalid file type" }, { status: 400 });
    }

    const buffer = Buffer.from(await file.arrayBuffer());
    const data = `data:${file.type};base64,${buffer.toString("base64")}`;

    const item = store.createMedia({
      filename: file.name,
      mimeType: file.type,
      size: file.size,
      data,
    });

    return NextResponse.json({ media: item }, { status: 201 });
  } catch {
    return NextResponse.json({ error: "Failed to upload file" }, { status: 500 });
  }
}
