-- What the event is for, and who it honours. The writing assistant and the
-- event page use these to fit the occasion. Existing events become 'other'.
ALTER TABLE events
    ADD COLUMN occasion TEXT NOT NULL DEFAULT 'other' CHECK (occasion IN (
        'birthday', 'wedding', 'anniversary', 'graduation', 'baby',
        'retirement', 'farewell', 'memorial', 'other'
    )),
    ADD COLUMN honoree_name TEXT;
