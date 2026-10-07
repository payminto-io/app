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
import { FormField, TextInput } from "@/components/ui/form-field";
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
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>New project</DialogTitle>
            <DialogDescription>An API key is generated with it and shown once.</DialogDescription>
          </DialogHeader>

          <FormField label="Name" htmlFor="ep-name">
            <TextInput id="ep-name" required value={name} onChange={(e) => setName(e.target.value)} placeholder="My store" />
          </FormField>

          <FormField label="Website" htmlFor="ep-website" hint="Optional">
            <TextInput
              id="ep-website"
              type="url"
              value={websiteURL}
              onChange={(e) => setWebsiteURL(e.target.value)}
              placeholder="https://example.com"
            />
          </FormField>

          <DialogFooter>
            <Button
              variant="outline"
              type="button"
              onClick={() => handleOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !name.trim()}>
              {isPending ? "Creating..." : "Create project"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
