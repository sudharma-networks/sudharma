import { render, screen } from "@testing-library/react";
import CommunityPage from "@/app/community/page";

test("shows the verified Sudharma Telegram destinations", () => {
  render(<CommunityPage />);

  expect(screen.getByRole("link", { name: /Official Telegram announcements/i })).toHaveAttribute(
    "href",
    "https://t.me/sudharmanetworks"
  );
  expect(screen.getByRole("link", { name: /Sudharma community discussion/i })).toHaveAttribute(
    "href",
    "https://t.me/sudharma_community"
  );
  expect(screen.getByText(/Only trust community links published on this website/i)).toBeInTheDocument();
});
