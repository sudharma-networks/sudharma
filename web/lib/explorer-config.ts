export const PUBLIC_EXPLORER_API_BASE_URL = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com";

export function resolveExplorerAPIBaseURL(configured?: string) {
  const override = configured?.trim();
  return override || PUBLIC_EXPLORER_API_BASE_URL;
}
