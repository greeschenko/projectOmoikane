import { Box, Typography } from "@mui/material";

export default function PublicFooter() {
  return (
    <Box
      component="footer"
      sx={{ bgcolor: "grey.100", py: 3, mt: "auto", textAlign: "center" }}
    >
      <Typography variant="body2" color="text.secondary">
        &copy; {new Date().getFullYear()} Omoikane
      </Typography>
    </Box>
  );
}
