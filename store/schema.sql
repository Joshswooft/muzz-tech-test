CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT UNIQUE,
	password TEXT,
	name TEXT,
	gender TEXT,
	dob TEXT,
	lat REAL DEFAULT 0,
	lng REAL DEFAULT 0
);

-- stores the user's swipes
CREATE TABLE IF NOT EXISTS swipes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	-- the person who performed the swipe
    swiper INTEGER REFERENCES users(id),
	-- the person getting 'swiped' on
    swipe_target INTEGER REFERENCES users(id),
	liked BOOLEAN,
	UNIQUE(swiper, swipe_target)
);

-- stores matches between 2 users
CREATE TABLE IF NOT EXISTS matches (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	-- user1 id is less than user2 id this is to maintain consistency
	user1 INTEGER REFERENCES users(id),
	user2 INTEGER REFERENCES users(id)
);

-- Create a trigger to enforce the constraint user1 < user2
CREATE TRIGGER IF NOT EXISTS enforce_user_order
BEFORE INSERT ON matches
FOR EACH ROW
WHEN NEW.user1 >= NEW.user2
BEGIN
    SELECT RAISE(ABORT, 'user1 must be less than user2');
END;

-- Create a trigger to automatically create match records if both users swiped on each other
CREATE TRIGGER IF NOT EXISTS create_match_trigger 
AFTER INSERT ON swipes
BEGIN
    INSERT INTO matches (user1, user2)
    SELECT
        (CASE WHEN NEW.swiper < NEW.swipe_target THEN NEW.swiper ELSE NEW.swipe_target END),
        (CASE WHEN NEW.swiper > NEW.swipe_target THEN NEW.swiper ELSE NEW.swipe_target END)
    WHERE NEW.liked = 1
    AND EXISTS (
        SELECT 1 FROM swipes
        WHERE swiper = NEW.swipe_target
        AND swipe_target = NEW.swiper
        AND liked = 1
    );
END;
