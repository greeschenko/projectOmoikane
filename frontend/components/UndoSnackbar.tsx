"use client";

import { Snackbar, Button, Alert } from "@mui/material";

interface UndoSnackbarProps {
  open: boolean;
  message: string;
  onUndo: () => void;
  onClose: () => void;
  autoHideDuration?: number;
}

export default function UndoSnackbar({ open, message, onUndo, onClose, autoHideDuration = 6000 }: UndoSnackbarProps) {
  return (
    <Snackbar
      open={open}
      autoHideDuration={autoHideDuration}
      onClose={onClose}
      anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
    >
      <Alert
        onClose={onClose}
        severity="info"
        variant="filled"
        action={
          <Button color="inherit" size="small" onClick={onUndo}>
            Undo
          </Button>
        }
        sx={{ width: "100%" }}
      >
        {message}
      </Alert>
    </Snackbar>
  );
}
