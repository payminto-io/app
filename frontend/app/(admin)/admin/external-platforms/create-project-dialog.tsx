"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

export function CreateProjectDialog({
  open,
  onOpenChange,
  onSubmit,
  isPending,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (input: { name: string; websiteURL?: string }) => void;
  isPending: boolean;
}) {
  const [name, setName] = useState("");
  const [websiteURL, setWebsiteURL] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit({ name, websiteURL: websiteURL || undefined });
  }

  function handleOpenChange(next: boolean) {
    if (!next) {
      setName("");
      setWebsiteURL("");
    }
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>New Project</DialogTitle>
            <DialogDescription>
              Create a new external platform project. An API key will be
              generated automatically.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="ep-name">Project Name</Label>
            <Input
              id="ep-name"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Store"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="ep-website">Website URL</Label>
            <Input
              id="ep-website"
              type="url"
              value={websiteURL}
              onChange={(e) => setWebsiteURL(e.target.value)}
              placeholder="https://example.com"
            />
            <p className="text-[11px] text-muted-foreground">Optional</p>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              type="button"
              onClick={() => handleOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !name.trim()}>
              {isPending ? "Creating..." : "Create Project"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
