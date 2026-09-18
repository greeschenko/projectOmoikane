"use client";

import { useEffect, useCallback, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import Placeholder from "@tiptap/extension-placeholder";
import Underline from "@tiptap/extension-underline";
import Link from "@tiptap/extension-link";
import TextAlign from "@tiptap/extension-text-align";
import {
  Box, ToggleButton, FormHelperText, Dialog, DialogTitle,
  DialogContent, IconButton, Card, CardMedia, Typography, Divider, Tooltip,
  Button, CircularProgress,
} from "@mui/material";
import FormatBoldIcon from "@mui/icons-material/FormatBold";
import FormatItalicIcon from "@mui/icons-material/FormatItalic";
import FormatUnderlinedIcon from "@mui/icons-material/FormatUnderlined";
import FormatStrikethroughIcon from "@mui/icons-material/FormatStrikethrough";
import ImageIcon from "@mui/icons-material/Image";
import FormatListBulletedIcon from "@mui/icons-material/FormatListBulleted";
import FormatListNumberedIcon from "@mui/icons-material/FormatListNumbered";
import FormatQuoteIcon from "@mui/icons-material/FormatQuote";
import CodeIcon from "@mui/icons-material/Code";
import LinkIcon from "@mui/icons-material/Link";
import HorizontalRuleIcon from "@mui/icons-material/HorizontalRule";
import UndoIcon from "@mui/icons-material/Undo";
import RedoIcon from "@mui/icons-material/Redo";
import CloseIcon from "@mui/icons-material/Close";
import TitleIcon from "@mui/icons-material/Title";
import FormatClearIcon from "@mui/icons-material/FormatClear";
import FormatAlignLeftIcon from "@mui/icons-material/FormatAlignLeft";
import FormatAlignCenterIcon from "@mui/icons-material/FormatAlignCenter";
import FormatAlignRightIcon from "@mui/icons-material/FormatAlignRight";
import FormatAlignJustifyIcon from "@mui/icons-material/FormatAlignJustify";
import UploadFileIcon from "@mui/icons-material/UploadFile";

interface MediaItem {
  id: string;
  filename: string;
  data: string;
  url?: string;
  thumbUrl?: string;
  alt?: string;
}

interface RichTextEditorProps {
  value: string;
  onChange: (html: string) => void;
  error?: boolean;
  helperText?: string;
  minimal?: boolean;
  placeholder?: string;
}

function ToolbarButton({
  active,
  onClick,
  icon,
  label,
  disabled,
}: {
  active?: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
  disabled?: boolean;
}) {
  return (
    <Tooltip title={label} placement="top">
      <ToggleButton
        value={label}
        selected={active}
        onChange={onClick}
        size="small"
        aria-label={label}
        disabled={disabled}
        sx={{ border: "none", borderRadius: 1, px: 1 }}
      >
        {icon}
      </ToggleButton>
    </Tooltip>
  );
}

export default function RichTextEditor({
  value,
  onChange,
  error,
  helperText,
  minimal = false,
  placeholder = "Start writing...",
}: RichTextEditorProps) {
  const [imageDialogOpen, setImageDialogOpen] = useState(false);
  const [mediaItems, setMediaItems] = useState<MediaItem[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: minimal ? false : { levels: [1, 2, 3] },
        codeBlock: minimal ? false : undefined,
        blockquote: minimal ? false : undefined,
        horizontalRule: minimal ? false : undefined,
        bulletList: minimal ? false : undefined,
        orderedList: minimal ? false : undefined,
        strike: minimal ? false : undefined,
      }),
      Image.configure({ inline: true, allowBase64: true }),
      Placeholder.configure({ placeholder }),
      Underline,
      ...(minimal ? [] : [
        Link.configure({
          openOnClick: false,
          HTMLAttributes: { class: "editor-link" },
        }),
        TextAlign.configure({ types: ["heading", "paragraph"] }),
      ]),
    ],
    content: value,
    editorProps: {
      attributes: {
        role: "textbox",
        "aria-label": "Content",
      },
    },
    onUpdate: useCallback(
      ({ editor: ed }: { editor: any }) => {
        onChange(ed.getHTML());
      },
      [onChange],
    ),
  });

  useEffect(() => {
    if (editor && value !== editor.getHTML()) {
      editor.commands.setContent(value);
    }
  }, [editor, value]);

  const openImageDialog = async () => {
    setImageDialogOpen(true);
    try {
      const res = await fetch("/api/media");
      const data = await res.json();
      setMediaItems(Array.isArray(data) ? data : (data.media ?? []));
    } catch {
      setMediaItems([]);
    }
  };

  const insertImage = (src: string, alt?: string) => {
    const attrs = alt ? { src, alt } : { src };
    editor?.chain().focus().setImage(attrs).run();
    setImageDialogOpen(false);
    setUploadError("");
  };

  // Upload a new file straight from the editor dialog, refresh the grid and
  // auto-insert the resulting image into the document.
  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setUploadError("");
    try {
      const body = new FormData();
      body.append("file", file);
      const res = await fetch("/api/media", { method: "POST", body });
      if (!res.ok) {
        setUploadError("Upload failed — please try again.");
        return;
      }
      const data = await res.json();
      const item: MediaItem | undefined = data?.media;
      if (!item) {
        setUploadError("Upload failed — no media returned.");
        return;
      }
      // Refresh the picker grid so the new file is selectable later too.
      setMediaItems((items) => [item, ...items.filter((m) => m.id !== item.id)]);
      const alt = window.prompt("Alt text (optional):", item.alt || "") || undefined;
      insertImage(item.url || item.data, alt);
    } catch {
      setUploadError("Upload failed — please try again.");
    } finally {
      setUploading(false);
      if (e.target) e.target.value = "";
    }
  };

  const addLink = () => {
    const url = window.prompt("Enter URL:");
    if (url) {
      editor?.chain().focus().setLink({ href: url }).run();
    }
  };

  if (!editor) return null;

  return (
    <Box>
      <Box
        sx={{
          border: 1,
          borderColor: error ? "error.main" : "divider",
          borderRadius: 1,
          overflow: "hidden",
        }}
      >
        <Box
          sx={{
            display: "flex",
            flexWrap: "wrap",
            alignItems: "center",
            gap: 0.25,
            p: 0.5,
            borderBottom: 1,
            borderColor: "divider",
            bgcolor: "grey.50",
          }}
        >
          <ToolbarButton
            active={editor.isActive("bold")}
            onClick={() => editor.chain().focus().toggleBold().run()}
            icon={<FormatBoldIcon fontSize="small" />}
            label="Bold"
          />
          <ToolbarButton
            active={editor.isActive("italic")}
            onClick={() => editor.chain().focus().toggleItalic().run()}
            icon={<FormatItalicIcon fontSize="small" />}
            label="Italic"
          />
          {!minimal && (
            <>
              <ToolbarButton
                active={editor.isActive("underline")}
                onClick={() => editor.chain().focus().toggleUnderline().run()}
                icon={<FormatUnderlinedIcon fontSize="small" />}
                label="Underline"
              />
              <ToolbarButton
                active={editor.isActive("strike")}
                onClick={() => editor.chain().focus().toggleStrike().run()}
                icon={<FormatStrikethroughIcon fontSize="small" />}
                label="Strikethrough"
              />
            </>
          )}

          <Divider orientation="vertical" flexItem sx={{ mx: 0.25 }} />

          {!minimal && (
            <>
              <ToolbarButton
                active={editor.isActive("heading", { level: 1 })}
                onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
                icon={<TitleIcon fontSize="small" sx={{ fontSize: "0.9rem" }} />}
                label="Heading 1"
              />
              <ToolbarButton
                active={editor.isActive("heading", { level: 2 })}
                onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
                icon={<TitleIcon fontSize="small" />}
                label="Heading 2"
              />
              <ToolbarButton
                active={editor.isActive("heading", { level: 3 })}
                onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
                icon={<TitleIcon fontSize="small" sx={{ fontSize: "0.75rem" }} />}
                label="Heading 3"
              />

              <Divider orientation="vertical" flexItem sx={{ mx: 0.25 }} />

              <ToolbarButton
                active={editor.isActive("bulletList")}
                onClick={() => editor.chain().focus().toggleBulletList().run()}
                icon={<FormatListBulletedIcon fontSize="small" />}
                label="Bullet List"
              />
              <ToolbarButton
                active={editor.isActive("orderedList")}
                onClick={() => editor.chain().focus().toggleOrderedList().run()}
                icon={<FormatListNumberedIcon fontSize="small" />}
                label="Ordered List"
              />
              <ToolbarButton
                active={editor.isActive("blockquote")}
                onClick={() => editor.chain().focus().toggleBlockquote().run()}
                icon={<FormatQuoteIcon fontSize="small" />}
                label="Blockquote"
              />
              <ToolbarButton
                active={editor.isActive("codeBlock")}
                onClick={() => editor.chain().focus().toggleCodeBlock().run()}
                icon={<CodeIcon fontSize="small" />}
                label="Code Block"
              />

              <Divider orientation="vertical" flexItem sx={{ mx: 0.25 }} />

              <ToolbarButton
                active={editor.isActive({ textAlign: "left" })}
                onClick={() => editor.chain().focus().setTextAlign("left").run()}
                icon={<FormatAlignLeftIcon fontSize="small" />}
                label="Align Left"
              />
              <ToolbarButton
                active={editor.isActive({ textAlign: "center" })}
                onClick={() => editor.chain().focus().setTextAlign("center").run()}
                icon={<FormatAlignCenterIcon fontSize="small" />}
                label="Align Center"
              />
              <ToolbarButton
                active={editor.isActive({ textAlign: "right" })}
                onClick={() => editor.chain().focus().setTextAlign("right").run()}
                icon={<FormatAlignRightIcon fontSize="small" />}
                label="Align Right"
              />
              <ToolbarButton
                active={editor.isActive({ textAlign: "justify" })}
                onClick={() => editor.chain().focus().setTextAlign("justify").run()}
                icon={<FormatAlignJustifyIcon fontSize="small" />}
                label="Justify"
              />

              <Divider orientation="vertical" flexItem sx={{ mx: 0.25 }} />

              <ToolbarButton
                active={editor.isActive("link")}
                onClick={addLink}
                icon={<LinkIcon fontSize="small" />}
                label="Insert Link"
              />
              <ToolbarButton
                onClick={() => editor.chain().focus().setHorizontalRule().run()}
                icon={<HorizontalRuleIcon fontSize="small" />}
                label="Horizontal Rule"
              />
            </>
          )}

          <ToolbarButton
            onClick={() => openImageDialog()}
            icon={<ImageIcon fontSize="small" />}
            label="Insert Image"
          />

          <Box sx={{ flex: 1 }} />

          <ToolbarButton
            onClick={() => editor.chain().focus().clearNodes().unsetAllMarks().run()}
            icon={<FormatClearIcon fontSize="small" />}
            label="Clear Formatting"
          />
          <ToolbarButton
            onClick={() => editor.chain().focus().undo().run()}
            icon={<UndoIcon fontSize="small" />}
            label="Undo"
            disabled={!editor.can().undo()}
          />
          <ToolbarButton
            onClick={() => editor.chain().focus().redo().run()}
            icon={<RedoIcon fontSize="small" />}
            label="Redo"
            disabled={!editor.can().redo()}
          />
        </Box>

        <Box
          sx={{
            p: 2,
            minHeight: minimal ? 80 : 300,
            cursor: "text",
            "& .tiptap": { outline: "none", minHeight: "inherit" },
          }}
          onClick={() => editor.commands.focus()}
        >
          <EditorContent editor={editor} />
        </Box>
      </Box>
      {helperText && (
        <FormHelperText error={error} sx={{ ml: 1 }}>
          {helperText}
        </FormHelperText>
      )}

      <Dialog
        open={imageDialogOpen}
        onClose={() => setImageDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          Select Image
          <IconButton size="small" onClick={() => setImageDialogOpen(false)}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 2 }}>
            <Button
              variant="contained"
              component="label"
              startIcon={<UploadFileIcon />}
              disabled={uploading}
              aria-label="Upload Image"
            >
              Upload Image
              <input type="file" hidden accept="image/*" onChange={handleUpload} />
            </Button>
            {uploading && <CircularProgress size={20} aria-label="Uploading" />}
            {uploadError && (
              <Typography color="error" variant="body2">
                {uploadError}
              </Typography>
            )}
          </Box>
          {mediaItems.length === 0 ? (
            <Typography
              color="text.secondary"
              sx={{ py: 4, textAlign: "center" }}
            >
              No images yet — upload one above, or from the Media page.
            </Typography>
          ) : (
            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
              {mediaItems.map((item) => (
                <Card
                  key={item.id}
                  sx={{
                    width: 120,
                    cursor: "pointer",
                    "&:hover": { opacity: 0.8 },
                  }}
                  onClick={() => {
                    const alt = window.prompt("Alt text (optional):", item.alt || "") || undefined;
                    insertImage(item.url || item.data, alt);
                  }}
                >
                  <CardMedia
                    component="img"
                    height={80}
                    image={item.thumbUrl || item.data}
                    alt={item.alt || item.filename}
                    sx={{ objectFit: "cover" }}
                  />
                  <Box sx={{ p: 0.5 }}>
                    <Typography variant="caption" noWrap>
                      {item.filename}
                    </Typography>
                  </Box>
                </Card>
              ))}
            </Box>
          )}
        </DialogContent>
      </Dialog>
    </Box>
  );
}
