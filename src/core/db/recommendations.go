/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package db

import (
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// maxRecommendHistory caps the number of remembered track keys per chat so
// the document (and dedup scan) stays small.
const maxRecommendHistory = 100

// RecommendHistory tracks which "similar songs" a chat has already been
// shown, keyed by chat ID, so repeat autoplay suggestions can be avoided.
type RecommendHistory struct {
	ID       int64    `bson:"_id"`
	TrackIDs []string `bson:"track_ids"`
}

// GetRecommendHistory returns the set of track keys ("platform:id") already
// recommended to a chat.
func (db *Database) GetRecommendHistory(chatID int64) []string {
	key := toKey(chatID)
	if cached, ok := db.recommendCache.Get(key); ok {
		return cached.TrackIDs
	}

	ctx, cancel := db.ctx()
	defer cancel()

	var hist RecommendHistory
	if err := db.recommendDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&hist); err != nil {
		return nil
	}

	db.recommendCache.Set(key, &hist)
	return hist.TrackIDs
}

// AddRecommendedTracks appends newly recommended track keys to a chat's
// history, capping the stored list at maxRecommendHistory entries (oldest
// first out).
func (db *Database) AddRecommendedTracks(chatID int64, trackKeys ...string) error {
	if len(trackKeys) == 0 {
		return nil
	}

	existing := db.GetRecommendHistory(chatID)
	seen := make(map[string]bool, len(existing))
	for _, k := range existing {
		seen[k] = true
	}

	merged := existing
	for _, k := range trackKeys {
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		merged = append(merged, k)
	}

	if len(merged) > maxRecommendHistory {
		merged = merged[len(merged)-maxRecommendHistory:]
	}

	ctx, cancel := db.ctx()
	defer cancel()

	_, err := db.recommendDB.UpdateOne(ctx, bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"track_ids": merged}},
		options.UpdateOne().SetUpsert(true))
	if err != nil {
		slog.Info("[DB] Failed to update recommendation history", "chat_id", chatID, "error", err)
		return err
	}

	db.recommendCache.Set(toKey(chatID), &RecommendHistory{ID: chatID, TrackIDs: merged})
	return nil
}

// HasRecommended reports whether a track key was already suggested to a chat.
func (db *Database) HasRecommended(chatID int64, trackKey string) bool {
	for _, k := range db.GetRecommendHistory(chatID) {
		if k == trackKey {
			return true
		}
	}
	return false
}
