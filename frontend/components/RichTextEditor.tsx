"use client";

import { useEffect, useCallback, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import {
  Box, ToggleButton, FormHelperText, Dialog, DialogTitle,
  DialogContent, IconButton, Card, CardMedia, Typography,
} from "@mui/material";
import FormatBoldIcon from "@mui/icons-material/FormatBold";
import FormatItalicIcon from "@mui/icons-material/FormatItalic";
import ImageIcon from "@mui/icons-material/Image";
import CloseIcon from "@mui/icons-material/Close";

interface MediaItem {
  id: string;
  filename: string;
  data: string;
}

interface RichTextEditorProps {
  value: string;
  onChange: (html: string) => void;
  error?: boolean;
  helperText?: string;
}

export default function RichTextEditor({ value, onChange, error, helperText }: RichTextEditorProps) {
  const [imageDialogOpen, setImageDialogOpen] = useState(false);
  const [mediaItems, setMediaItems] = useState<MediaItem[]>([]);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Image.configure({ inline: true }),
    ],
    content: value,
    editorProps: {
      attributes: {
        role: "textbox",
        "aria-label": "Content",
      },
    },
    onUpdate: useCallback(
      ({ editor: ed }) => { onChange(ed.getHTML()); },
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
      setMediaItems(data.media ?? []);
    } catch {
      setMediaItems([]);
    }
  };

  const insertImage = (src: string) => {
    editor?.chain().focus().setImage({ src }).run();
    setImageDialogOpen(false);
  };

  if (!editor) return null;

  return (
    <Box>
      <Box sx={{ border: 1, borderColor: error ? "error.main" : "divider", borderRadius: 1 }}>
        <Box sx={{ display: "flex", gap: 0.5, p: 0.5, borderBottom: 1, borderColor: "divider", bgcolor: "grey.50" }}>
          <ToggleButton
            value="bold"
            selected={editor.isActive("bold")}
            onChange={() => editor.chain().focus().toggleBold().run()}
            size="small"
            aria-label="Bold"
            sx={{ border: "none", borderRadius: 1 }}
          >
            <FormatBoldIcon fontSize="small" />
          </ToggleButton>
          <ToggleButton
            value="italic"
            selected={editor.isActive("italic")}
            onChange={() => editor.chain().focus().toggleItalic().run()}
            size="small"
            aria-label="Italic"
            sx={{ border: "none", borderRadius: 1 }}
          >
            <FormatItalicIcon fontSize="small" />
          </ToggleButton>
          <ToggleButton
            value="image"
            selected={false}
            onChange={() => openImageDialog()}
            size="small"
            aria-label="Insert Image"
            sx={{ border: "none", borderRadius: 1 }}
          >
            <ImageIcon fontSize="small" />
          </ToggleButton>
        </Box>
        <Box sx={{ p: 2, minHeight: 120, cursor: "text" }} onClick={() => editor.commands.focus()}>
          <EditorContent editor={editor} />
        </Box>
      </Box>
      {helperText && (
        <FormHelperText error={error} sx={{ ml: 1 }}>
          {helperText}
        </FormHelperText>
      )}

      <Dialog open={imageDialogOpen} onClose={() => setImageDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          Select Image
          <IconButton size="small" onClick={() => setImageDialogOpen(false)}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          {mediaItems.length === 0 ? (
            <Typography color="text.secondary" sx={{ py: 4, textAlign: "center" }}>
              No images uploaded yet. Upload media from the Media page.
            </Typography>
          ) : (
            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
              {mediaItems.map((item) => (
                <Card
                  key={item.id}
                  sx={{ width: 120, cursor: "pointer", "&:hover": { opacity: 0.8 } }}
                  onClick={() => insertImage(item.data)}
                >
                  <CardMedia component="img" height={80} image={item.data} alt={item.filename} sx={{ objectFit: "cover" }} />
                  <Box sx={{ p: 0.5 }}>
                    <Typography variant="caption" noWrap>{item.filename}</Typography>
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
